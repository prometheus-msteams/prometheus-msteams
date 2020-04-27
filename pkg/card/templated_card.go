package card

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/alertmanager/template"
	"go.opencensus.io/trace"
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

// msTeamsCard divides a MS Teams card into two parts:
// 	* Sections: contains the actual payload - we use sections to divide a too large message/card into multiple cards
//  * EverythingElse: contains everything else that is required for a valid MS Teams card, like @type, @context, summary etc. 
// more information see https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference#example-office-365-connector-card
type msTeamsCard struct {
	Sections       []map[string]interface{} `json:"sections"`
	EverythingElse map[string]interface{}   `json:"-"`
}

// Constants for creating a Card
const (
	// Maximum message size of 14336 Bytes (14KB)
	maxSize = 14336
	// Maximum number of sections
	// ref: https://docs.microsoft.com/en-us/microsoftteams/platform/concepts/cards/cards-reference#notes-on-the-office-365-connector-card
	maxCardSections = 10
)

func (m *templatedCard) Convert(ctx context.Context, promAlert webhook.Message) (JSON, error) {
	_, span := trace.StartSpan(ctx, "templatedCard.Convert")
	defer span.End()

	totalMessage, err := m.executeTemplate(promAlert)
	if err != nil {
		return nil, err
	}

	cards, err := m.createFinalCards(totalMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to create final cards: %w", err)
	}

	cardb, err := json.Marshal(cards)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal final cards: %w", err)
	}
	span.Annotate(
		[]trace.Attribute{
			trace.StringAttribute("card", string(cardb)),
		},
		"card created",
	)

	return cards, nil
}

func (m *templatedCard) executeTemplate(promAlert webhook.Message) (string, error) {
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

	totalMessage, err := m.template.ExecuteTextString(
		`{{ template "teams.card" . }}`, data,
	)
	if err != nil {
		return "", fmt.Errorf("failed to template alerts: %w", err)
	}
	return totalMessage, nil
}

func (m *templatedCard) createFinalCards(totalMessage string) (JSON, error) {
	compactTotalMessage, err := compact([]byte(totalMessage))
	if err != nil {
		return nil, fmt.Errorf("failed to compact message: %w", err)
	}
	sizeMessage := len(compactTotalMessage)

	card, err := unmarshalMSTeamsCard(totalMessage)
	if err != nil {
		return nil, err
	}

	var cards JSON
	if len(card.Sections) > maxCardSections {
		cards, err := m.splitSections(card)
		if err != nil {
			return nil, fmt.Errorf("failed to split message: %w", err)
		}
		return cards, nil
	}
	if sizeMessage > maxSize {
		cards, err = m.splitLargeMessage(card)
		if err != nil {
			return nil, fmt.Errorf("failed to split message: %w", err)
		}
		return cards, nil
	}

	err = json.Unmarshal([]byte("["+totalMessage+"]"), &cards)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal full message: %w", err)
	}
	return cards, nil
}

func (m *templatedCard) splitSections(card msTeamsCard) (JSON, error) {
	var v JSON
	startIndex := 0
	allSectionsProcessed := false
	for !allSectionsProcessed {
		// get the maximum allowed number of sections
		endIndex := min(startIndex+maxCardSections, len(card.Sections))
		tmpSections := card.Sections[startIndex:endIndex]

		// construct a complete MS Teams card with sections and everything else
		card.EverythingElse["sections"] = tmpSections
		cardb, err := json.Marshal(card.EverythingElse)
		if err != nil {
			panic(err)
		}
		var vtmp map[string]interface{}
		err = json.Unmarshal(cardb, &vtmp)
		if err != nil {
			panic(err)
		}
		// TODO: verify that vtmp is not too large in size using splitLargeMessage()

		v = append(v, vtmp)

		// reset all values for the next loop
		delete(card.EverythingElse, "sections")
		startIndex = endIndex
		if endIndex >= len(card.Sections) {
			allSectionsProcessed = true
		}
	}
	return v, nil
}

func (m *templatedCard) splitLargeMessage(card msTeamsCard) (JSON, error) {
	var v JSON
	// TODO: implement
	return v, nil
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func compact(data []byte) (string, error) {
	buffer := new(bytes.Buffer)
	err := json.Compact(buffer, data)
	if err != nil {
		return "", fmt.Errorf("Error calling json.Compact: %w", err)
	}
	return buffer.String(), nil
}

func unmarshalMSTeamsCard(totalMessage string) (msTeamsCard, error) {
	var card msTeamsCard
	err := json.Unmarshal([]byte(totalMessage), &card)
	if err != nil {
		return msTeamsCard{}, fmt.Errorf("failed to unmarshal totalMessage: %w", err)
	}
	if err := json.Unmarshal([]byte(totalMessage), &card.EverythingElse); err != nil {
		return msTeamsCard{}, fmt.Errorf("failed to unmarshal to card.EverythingElse: %w", err)
	}
	delete(card.EverythingElse, "sections")

	return card, nil
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

// ParseTemplateFile creates an alertmanager template from the given file.
func ParseTemplateFile(f string) (*template.Template, error) {
	funcs := template.DefaultFuncs
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
		return nil, fmt.Errorf("Failed to parse template: %v: %v", err, err)
	}

	return tmpl, nil
}
