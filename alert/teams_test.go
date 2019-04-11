package alert

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/template"
	log "github.com/sirupsen/logrus"
)

func jsonparserGetString(data []byte, key string) string {
	val, _, _, _ := jsonparser.Get(data, key)
	return string(val)
}

func createTestCards(p notify.WebhookMessage, templateFile string) (string, error) {
	funcs := template.DefaultFuncs
	funcs["counter"] = func() func() int {
		i := -1
		return func() int {
			i++
			return i
		}
	}
	template.DefaultFuncs = funcs
	tmpl, err := template.FromGlobs(templateFile)
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

	cards, err := CreateCards(p, webhook)
	return cards, err
}
func createCardsFromPrometheusTestAlert(testdata string, templateFile string, t *testing.T) (string, error) {
	var p notify.WebhookMessage
	b, err := ioutil.ReadFile(testdata)
	if err != nil {
		t.Fatalf("Failed reading file %s got error: +%v", testdata, err)
	}
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatalf("Failed unmarshalling testdata file %s, got error: +%v",
			testdata, err)
	}
	return createTestCards(p, templateFile)
}

func TestQuerySections(t *testing.T) {
	validMessage := "{\"@type\":\"MessageCard\",\"@context\":\"http://schema.org/extensions\",\"themeColor\":\"FFA500\",\"summary\":\"Server High Memory usage\",\"title\":\"Prometheus Alert (firing)\",\"sections\":[{\"activityTitle\":\"[10.80.40.11 reported high memory usage with 23.28%.](http://docker.for.mac.host.internal:9093)\",\"facts\":[{\"name\":\"description\",\"value\":\"10.80.40.11 reported high memory usage with 23.28%.\"},{\"name\":\"summary\",\"value\":\"Server High Memory usage\"},{\"name\":\"alertname\",\"value\":\"high\\_memory\\_load\"},{\"name\":\"instance\",\"value\":\"instance-with-hyphen\\_and\\_underscore\"},{\"name\":\"job\",\"value\":\"docker\\_nodes\"},{\"name\":\"monitor\",\"value\":\"master\"},{\"name\":\"severity\",\"value\":\"warning\"}],\"markdown\":true}]}"
	_, err := querySections(validMessage)
	if err != nil {
		t.Fatalf("Not possible to query message with key 'sections': %s", err)
	}
}

func TestCreateCards(t *testing.T) {
	testdata := "testdata/prom_post_request.json"
	cards, _ := createCardsFromPrometheusTestAlert(testdata, "../default-message-card.tmpl", t)

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

	// test that 2 alerts get combined to one message
	testdata = "testdata/prom_post_request_2_alerts.json"
	cards, _ = createCardsFromPrometheusTestAlert(testdata, "../default-message-card.tmpl", t)

	length = 0
	jsonparser.ArrayEach([]byte(cards), func(card []byte, dataType jsonparser.ValueType, offset int, err error) {
		length++
	})
	if length != 1 {
		t.Fatalf("CreateCards error: should create 1 card, got %d cards", length)
	}
}

func TestCreateCardsTemplateWithoutSections(t *testing.T) {
	testdata := "testdata/prom_post_request.json"
	errorMessage := "Failed to parse json with key 'sections': Key path not found"
	_, err := createCardsFromPrometheusTestAlert(testdata, "testdata/message-card-without-sections.tmpl", t)
	if (err == nil) || (err.Error() != errorMessage) {
		t.Fatalf("CreateCards should produce error: '%v', got '%v'", errorMessage, err)
	}
}

func TestTemplateWithoutSummaryOrText(t *testing.T) {
	testdata := "testdata/prom_post_request_without_common_summary.json"
	cards, _ := createCardsFromPrometheusTestAlert(testdata, "../default-message-card.tmpl", t)
	jsonparser.ArrayEach([]byte(cards), func(card []byte, dataType jsonparser.ValueType, offset int, err error) {
		summary := jsonparserGetString(card, "summary")
		text := jsonparserGetString(card, "text")
		if (summary == "") && (text == "") {
			t.Fatalf("Microsoft Teams message requires Summary or Text")
		}
	})
}

func TestLargePostRequest(t *testing.T) {
	// test larged sized message
	testdata := "testdata/large_prom_post_request.json"
	cards, _ := createCardsFromPrometheusTestAlert(testdata, "../default-message-card.tmpl", t)

	length := 0
	jsonparser.ArrayEach([]byte(cards), func(card []byte, dataType jsonparser.ValueType, offset int, err error) {
		length++
	})
	if length != 2 {
		t.Fatalf("Too Large sized Message error: should create 2 cards, got %d cards", length)
	}

	// test too many alerts which results in too many sections
	testdata = "testdata/prom_post_request_12_alerts.json"
	cards, _ = createCardsFromPrometheusTestAlert(testdata, "../default-message-card.tmpl", t)

	length = 0
	jsonparser.ArrayEach([]byte(cards), func(card []byte, dataType jsonparser.ValueType, offset int, err error) {
		length++
	})
	if length != 2 {
		t.Fatalf("Too many Sections error: should create 2 cards, got %d cards", length)
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
		cards, _ := createTestCards(p, "../default-message-card.tmpl")
		jsonparser.ArrayEach([]byte(cards), func(card []byte, dataType jsonparser.ValueType, offset int, err error) {
			got := jsonparserGetString(card, "themeColor")
			if got != tc.wantColor {
				t.Fatalf("Failed assigning themes color to card: got %s, want %s",
					got, tc.wantColor)
			}
		})
	}
}

// TestAlertsSectionsOrdering tests https://github.com/bzon/prometheus-msteams/issues/38
func TestAlertsSectionsOrdering(t *testing.T) {
	testdata := "testdata/prom_post_request.json"
	cards, _ := createCardsFromPrometheusTestAlert(testdata, "../default-message-card.tmpl", t)
	facts, _, _, _ := jsonparser.Get([]byte(cards), "[0]", "sections", "[0]", "facts")
	i := 0
	jsonparser.ArrayEach(facts, func(fact []byte, dataType jsonparser.ValueType, offset int, err error) {
		key, _, _, _ := jsonparser.Get(fact, "name")
		switch i {
		case 0:
			if string(key) != "description" {
				t.Fatalf("Alert out of order: got %s, want %s", string(key), "description")
			}
			i++
		case 1:
			if string(key) != "summary" {
				t.Fatalf("Alert out of order: got %s, want %s", string(key), "summary")
			}
			i++
		case 2:
			if string(key) != "alertname" {
				t.Fatalf("Alert out of order: got %s, want %s", string(key), "alertname")
			}
			i++
		case 3:
			if string(key) != "instance" {
				t.Fatalf("Alert out of order: got %s, want %s", string(key), "instance")
			}
			i++
		case 4:
			if string(key) != "job" {
				t.Fatalf("Alert out of order: got %s, want %s", string(key), "job")
			}
			i++
		case 5:
			if string(key) != "monitor" {
				t.Fatalf("Alert out of order: got %s, want %s", string(key), "monitor")
			}
			i++
		case 6:
			if string(key) != "severity" {
				t.Fatalf("Alert out of order: got %s, want %s", string(key), "severity")
			}
			i++
		}
	})
}

func TestSendCard200Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	resp, _ := SendCard(ts.URL, "somecard", 10, 10, 10)
	if resp.StatusCode != 200 {
		t.Fatalf("Response status code not 200: got %s, want: 200", string(resp.StatusCode))
	}
}

func TestSendCard404Failure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer ts.Close()

	_, errSendCard := SendCard(ts.URL, "somecard", 10, 10, 10)
	matched, _ := regexp.MatchString("404", errSendCard.Error())
	if matched == false {
		t.Fatalf("Response status code not 404: want: 404, error string: \"%v\"", errSendCard)
	}
}

func TestSendCardInvalidWebhookProto(t *testing.T) {
	_, err := SendCard("somewebhook", "somecard", 10, 10, 10)
	if err == nil {
		t.Fatal("Error was meant to be thrown for invalid protocol")
		t.FailNow()
	}
}
