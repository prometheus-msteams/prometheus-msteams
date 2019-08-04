package alert

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"github.com/stretchr/testify/assert"

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

func TestConcatinatingKeyValue(t *testing.T) {
	tt := []struct {
		key  string
		val string
		want string
	}{
		{"key", "val", "\"key\":\"val\""},
		{"key", "[val]", "\"key\":\"[val]\""},
		{"key", "[{\"key2\":\"val2\"}]", "\"key\":[{\"key2\":\"val2\"}]"},
	}

	for _, tc := range tt {
		got := concatKeyValue(tc.key, tc.val)
		assert.Equal(t, tc.want, got)
	}
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
	assert.Equal(t, 1, length, "CreateCards error: should create 1 card")

	jsonparser.ArrayEach([]byte(cards), func(card []byte, dataType jsonparser.ValueType, offset int, err error) {
		want := "FFA500"
		got := jsonparserGetString(card, "themeColor")
		assert.Equal(t, want, got, "CreateCards error")

		want = "Server High Memory usage"
		got = jsonparserGetString(card, "summary")
		assert.Equal(t, want, got, "CreateCards error")
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
	assert.Equal(t, 1, length, "CreateCards error: should create 1 card")
}

func TestCreateCardsTemplateWithoutSections(t *testing.T) {
	testdata := "testdata/prom_post_request.json"
	errorMessage := "Failed to parse json with key 'sections': Key path not found"
	_, err := createCardsFromPrometheusTestAlert(testdata, "testdata/message-card-without-sections.tmpl", t)
	assert.NotNil(t, err, "CreateCards should produce error")
	assert.Equal(t, errorMessage, err.Error(), "CreateCards should produce correct error message")
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
	assert.Equal(t, 2, length, "Too Large sized Message error: should create 2 cards")

	// test too many alerts which results in too many sections
	testdata = "testdata/prom_post_request_12_alerts.json"
	cards, _ = createCardsFromPrometheusTestAlert(testdata, "../default-message-card.tmpl", t)

	length = 0
	jsonparser.ArrayEach([]byte(cards), func(card []byte, dataType jsonparser.ValueType, offset int, err error) {
		length++
	})
	assert.Equal(t, 2, length, "Too many Sections error: should create 2 cards")
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
			assert.Equal(t, tc.wantColor, got, "Failed assigning themes color to card")
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
			assert.Equal(t, "description", string(key), "Alert out of order")
			i++
		case 1:
			assert.Equal(t, "summary", string(key), "Alert out of order")
			i++
		case 2:
			assert.Equal(t, "alertname", string(key), "Alert out of order")
			i++
		case 3:
			assert.Equal(t, "instance", string(key), "Alert out of order")
			i++
		case 4:
			assert.Equal(t, "job", string(key), "Alert out of order")
			i++
		case 5:
			assert.Equal(t, "monitor", string(key), "Alert out of order")
			i++
		case 6:
			assert.Equal(t, "severity", string(key), "Alert out of order")
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
	assert.Equal(t, 200, resp.StatusCode, "Response status code not 200")
}

func TestSendCard404Failure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer ts.Close()

	resp, _ := SendCard(ts.URL, "somecard", 10, 10, 10)
	assert.Equal(t, 404, resp.StatusCode, "Response status code not 404")
}

func TestSendCardInvalidWebhookProto(t *testing.T) {
	_, err := SendCard("somewebhook", "somecard", 10, 10, 10)
	assert.NotNil(t, err, "Error was meant to be thrown for invalid protocol")
}
