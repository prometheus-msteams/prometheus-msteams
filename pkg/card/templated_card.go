package card

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/alertmanager/template"
	"go.opencensus.io/trace"
	"k8s.io/helm/pkg/engine"
)

// templatedCard implements Converter using Alert manager templating.
type templatedCard struct {
	template *template.Template
	// If true, replace all character `_` with `\\_` in the prometheus alert.
	escapeUnderscores bool
}

// NewTemplatedCardCreator creates a templatedCard.
func NewTemplatedCardCreator(template *template.Template, escapeUnderscores bool) Converter {
	return &templatedCard{template, escapeUnderscores}
}

func (m *templatedCard) Convert(ctx context.Context, promAlert webhook.Message) (Office365ConnectorCard, error) {
	_, span := trace.StartSpan(ctx, "templatedCard.Convert")
	defer span.End()

	cardString, err := m.executeTemplate(promAlert)
	if err != nil {
		return Office365ConnectorCard{}, err
	}

	var card Office365ConnectorCard
	if err := json.Unmarshal([]byte(cardString), &card); err != nil {
		return Office365ConnectorCard{}, err
	}

	if card.Type != "MessageCard" {
		return Office365ConnectorCard{}, errors.New("only MessageCard type is supported")
	}

	return card, nil
}

func (m *templatedCard) executeTemplate(promAlert webhook.Message) (string, error) {
	// TODO(bzon): Maybe we can escape underscores after the office 365 card is finally created?
	// That approach would be simpler to read and probably a performance gain because
	// we don't have to run json.NewEncoder(v).Encode() multiple times.
	if m.escapeUnderscores {
		promAlert = jsonEscapeMessage(promAlert)
	}

	data := &template.Data{
		Receiver:          promAlert.Receiver,
		Status:            promAlert.Status,
		Alerts:            promAlert.Alerts,
		GroupLabels:       promAlert.GroupLabels,
		CommonLabels:      promAlert.CommonLabels,
		CommonAnnotations: promAlert.CommonAnnotations,
		ExternalURL:       promAlert.ExternalURL,
	}

	cardString, err := m.template.ExecuteTextString(
		`{{ template "teams.card" . }}`, data,
	)
	if err != nil {
		return "", fmt.Errorf("failed to template alerts: %w", err)
	}

	return cardString, nil
}

func jsonEncode(str string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	err := enc.Encode(str)
	if err != nil {
		return ""
	}
	return string(buf.Bytes()[1 : len(buf.Bytes())-2])
}

// json escape all string values in kvData and also escape
// '_' char so it does not get processed as markdown italic
func jsonEncodeAlertmanagerKV(kvData template.KV) {
	for k, v := range kvData {
		kvData[k] = strings.ReplaceAll(jsonEncode(v), `_`, `\\_`)
	}
}

func jsonEscapeMessage(promAlert webhook.Message) webhook.Message {
	retPromAlert := promAlert
	jsonEncodeAlertmanagerKV(retPromAlert.GroupLabels)
	jsonEncodeAlertmanagerKV(retPromAlert.CommonLabels)
	jsonEncodeAlertmanagerKV(retPromAlert.CommonAnnotations)
	for _, alert := range retPromAlert.Alerts {
		jsonEncodeAlertmanagerKV(alert.Labels)
		jsonEncodeAlertmanagerKV(alert.Annotations)
	}
	return retPromAlert
}

/* ParseTemplateFile creates an alertmanager template from the given file.
The functions include all functions (except 'env' and 'expandenv' ) from sprig (http://masterminds.github.io/sprig/)
and the following functions from HELM templating:
  - toToml
  - toYaml
  - fromYaml
  - toJson
  - fromJson
*/
func ParseTemplateFile(f string) (*template.Template, error) {
	funcs := template.DefaultFuncs
	for k, v := range engine.FuncMap() {
		funcs[k] = v
	}
	funcs["counter"] = func() func() int {
		i := -1
		return func() int {
			i++
			return i
		}
	}
	template.DefaultFuncs = funcs

	if _, err := os.Stat(f); os.IsNotExist(err) {
		return nil, fmt.Errorf("template file %s does not exist", f)
	}

	tmpl, err := template.FromGlobs(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %v: %v", err, err)
	}

	return tmpl, nil
}
