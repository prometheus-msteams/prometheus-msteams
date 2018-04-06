package alert

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAlertManagerHandler(t *testing.T) {
	os.Setenv("MARKDOWN_ENABLED", "yes")
	testTable := []struct {
		name           string
		promAlertJSON  string
		postStatusCode int
		teams          Teams
	}{
		{
			name:           "send a teams card using an invalid route must return 404",
			promAlertJSON:  "examples/prom_post_request.json",
			postStatusCode: http.StatusNotFound,
			teams: Teams{
				RequestURI: "/teams_404",
				WebhookURL: "https://outlook.office.com/webhook/xxxx",
			},
		},
		{
			name:           "send a teams card using an empty webhook must return 500",
			promAlertJSON:  "examples/prom_post_request.json",
			postStatusCode: http.StatusInternalServerError,
			teams: Teams{
				RequestURI: "/teams_ok_but_config_err",
			},
		},
		{
			name:           "send a teams card using correct webhook must return 200",
			promAlertJSON:  "examples/prom_post_request.json",
			postStatusCode: http.StatusOK,
			teams: Teams{
				RequestURI: "/teams_ok",
				WebhookURL: os.Getenv("GOTEST_TEAMS_INCOMING_WEBHOOK_URL"),
			},
		},
	}

	for _, tc := range testTable {
		mux := http.NewServeMux()
		// Valid routes
		mux.HandleFunc("/teams_ok_but_config_err", tc.teams.PrometheusAlertManagerHandler)
		mux.HandleFunc("/teams_ok", tc.teams.PrometheusAlertManagerHandler)
		ts := httptest.NewServer(mux)
		defer ts.Close()

		promAlertInBytes, err := ioutil.ReadFile(tc.promAlertJSON)
		if err != nil {
			t.Fatal(err)
		}

		t.Run(tc.name, func(t *testing.T) {
			promBuffer := bytes.NewBuffer(promAlertInBytes)
			res, err := http.Post(ts.URL+tc.teams.RequestURI, "application/json", promBuffer)
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != tc.postStatusCode {
				t.Fatalf("Want status code %d; got %d", tc.postStatusCode, res.StatusCode)
			}
		})
	}

}
