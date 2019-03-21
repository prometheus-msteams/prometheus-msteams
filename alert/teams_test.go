package alert

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/template"
	log "github.com/sirupsen/logrus"
)

func jsonparserGetString(data []byte, key string) string {
	get, _, _, _ := jsonparser.Get(data, key)
	return string(get)
}

func createTestCards(p notify.WebhookMessage) string {
	funcs := template.DefaultFuncs
	funcs["counter"] = func() func() int {
		i := -1
		return func() int {
			i++
			return i
		}
	}
	template.DefaultFuncs = funcs
	tmpl, err := template.FromGlobs("../default-message-card.tmpl")
	if err != nil {
		log.Errorf("Failed to parse template: %v", err)
		os.Exit(1)
	}
	var webhook *PrometheusWebhook
	webhook = &PrometheusWebhook{
		RequestURI:      "/alertmanager",
		TeamsWebhookURL: "https://outlook.office.com/webhook/xxxxxxx/IncomingWebhook/yyyyyyyy/zzzzzzz",
		MarkdownEnabled: true,
		Template:        tmpl,
	}

	cards, _ := CreateCards(p, webhook)
	return cards
}
func createCardsFromPrometheusTestAlert(testdata string, t *testing.T) string {
	var p notify.WebhookMessage
	b, err := ioutil.ReadFile(testdata)
	if err != nil {
		t.Fatalf("Failed reading file %s got error: +%v", testdata, err)
	}
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatalf("Failed unmarshalling testdata file %s, got error: +%v",
			testdata, err)
	}
	return createTestCards(p)
}

func TestCreateCards(t *testing.T) {
	testdata := "testdata/prom_post_request.json"
	cards := createCardsFromPrometheusTestAlert(testdata, t)

	length := 0
	jsonparser.ArrayEach([]byte(cards), func(card []byte, dataType jsonparser.ValueType, offset int, err error) {
		length++
	})
	if length != 1 {
		t.Fatalf("CreateCards error: should create 1 card, got %d cards", length)
	}

	jsonparser.ArrayEach([]byte(cards), func(card []byte, dataType jsonparser.ValueType, offset int, err error) {
		want := "FFA500"
		got := jsonparserGetString(card, "themeColor")
		if got != want {
			t.Fatalf("CreateCards error: got %s, want %s", got, want)
		}

		want = "Server High Memory usage"
		got = jsonparserGetString(card, "summary")
		if got != want {
			t.Fatalf("CreateCards error: got %s, want %s", got, want)
		}
	})
}

func TestLargePostRequest(t *testing.T) {
	testdata := "testdata/large_prom_post_request.json"
	cards := createCardsFromPrometheusTestAlert(testdata, t)

	length := 0
	jsonparser.ArrayEach([]byte(cards), func(card []byte, dataType jsonparser.ValueType, offset int, err error) {
		length++
	})
	if length != 2 {
		t.Fatalf("TeamsMessageCard.CreatedCard error: should create 2 cards, got %d cards", length)
	}
}

func TestStatusColorFiring(t *testing.T) {
	tt := []struct {
		severity  string
		wantColor string
	}{
		{"warning", "FFA500"},
		{"critical", "8C1A1A"},
		{"unknown", "808080"},
	}

	for _, tc := range tt {
		data := &template.Data{Status: "firing", CommonLabels: map[string]string{"severity": tc.severity}}
		p := notify.WebhookMessage{Data: data}
		cards := createTestCards(p)
		jsonparser.ArrayEach([]byte(cards), func(card []byte, dataType jsonparser.ValueType, offset int, err error) {
			got := jsonparserGetString(card, "themeColor")
			if got != tc.wantColor {
				t.Fatalf("Failed assigning themes color to card: got %s, want %s",
					got, tc.wantColor)
			}
		})
	}
}
