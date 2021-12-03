package main

import (
	_ "net/http/pprof"
	"os"
	"io"
	"testing"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	
	"github.com/prometheus-msteams/prometheus-msteams/pkg/transport"
	"github.com/go-kit/kit/log"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func Test_validateWebhook(t *testing.T) {
	type args struct {
		u string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "legacy hook", args: args{u: "https://outlook.office.com/webhook/1e21eb6d-60f4-432f-9428-82cf86ec55a3@b12d4011-2ea0-4377-a99b-35c565546afd/IncomingWebhook/091d20cd2a594db0b2d7ed2937d7bd6d/91f7b1cc-96d0-4612-bedd-8b820c869464"}, wantErr: false},
		{name: "new webhook", args: args{u: "https://example.webhook.office.com/webhookb2/5c51ab94-86c0-4ba3-a66c-c2ad73acc531@e3ebcd52-aa57-25e8-a214-94fb325450f4/IncomingWebhook/9f226e3c36fb47249f14d4dab2d5b845/92bac26e-62c5-427f-ac91-e51a268f94ca"}, wantErr: false},
		{name: "missing https", args: args{u: "outlook.office.com/webhook/xxxx/xxxx"}, wantErr: true},
		{name: "only http", args: args{u: "http://outlook.office.com/webhook/xxxx/xxxx"}, wantErr: true},
		{name: "https but invalid", args: args{u: "https://example.com"}, wantErr: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if err := validateWebhook(tt.args.u); (err != nil) != tt.wantErr {
				t.Errorf("validateWebhook() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_reloadHandler(t *testing.T) {

	// Setup variables
	httpClient := http.DefaultClient
	validateURL := false
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	tc := PromTeamsConfig{
		Connectors: []map[string]string{
			{
				"connector1": "https://localhost:8080",
			},
		},
		ConnectorsWithCustomTemplates: []ConnectorWithCustomTemplate{},
	}

	// Create the initial routes for our server
	routes, err := setupServices(tc, nil, httpClient, &validateURL, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Configure handlers
	handler := transport.NewServer(logger, routes, nil)
	handler.POST("/reload", func(c echo.Context) error {
		routes, err = setupServices(tc, nil, httpClient, &validateURL, logger)
		if err != nil {
			logger.Log("err", err)
			return c.JSON(500, err)
		}
		transport.ReloadRoutes(handler, routes, logger)
		return c.JSON(200, tc.Connectors)
	})
	handler.GET("/config", func(c echo.Context) error {
		return c.JSON(200, tc.Connectors)
	})

	// Call GET on /connector2 and check that we get a 404 not found. 
	// Should return 404 not found because we haven't created it yet
	resp, body := testURL(handler, "/connector2", http.MethodGet, nil)
	assert.Equal(t, "{\"message\":\"Not Found\"}\n", string(body))
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Now add connector2 to the map array and reload the services by calling POST on /reload
	// Should return 200 OK 
	tc.Connectors = append(tc.Connectors, map[string]string{
		"connector2": "https://localhost:8443",
	})
	resp, body = testURL(handler, "/reload", http.MethodPost, nil)
	assert.Equal(t, "[{\"connector1\":\"https://localhost:8080\"},{\"connector2\":\"https://localhost:8443\"}]\n", string(body))
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// The reload handler should now have reloaded all routes and we should be able to call GET on /connector2
	// We expect to get 405 method not allowed since the handler expects POST. But 405 tells us that the connector now exists in the server
	resp, body = testURL(handler, "/connector2", http.MethodGet, nil)
	assert.Equal(t, "{\"message\":\"Method Not Allowed\"}\n", string(body))
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	// Also call /config to ensure that it returns an expected list of connectors
	resp, body = testURL(handler, "/config", http.MethodGet, nil)
	assert.Equal(t, "[{\"connector1\":\"https://localhost:8080\"},{\"connector2\":\"https://localhost:8443\"}]\n", string(body))
	assert.Equal(t, http.StatusOK, resp.StatusCode)

}

func testURL(handler http.Handler, url, method string, data io.Reader) (*http.Response, []byte) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, url, data)
	handler.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return w.Result(), body
}