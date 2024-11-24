package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"testing"

	"github.com/prometheus-msteams/prometheus-msteams/pkg/service"
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
		{name: "legacy hook", args: args{u: "https://example.webhook.office.com/webhookb2/1e21eb6d-60f4-432f-9428-82cf86ec55a3@b12d4011-2ea0-4377-a99b-35c565546afd/IncomingWebhook/091d20cd2a594db0b2d7ed2937d7bd6d/91f7b1cc-96d0-4612-bedd-8b820c869464"}, wantErr: false},
		{name: "new webhook", args: args{u: "https://example.webhook.office.com/webhookb2/5c51ab94-86c0-4ba3-a66c-c2ad73acc531@e3ebcd52-aa57-25e8-a214-94fb325450f4/IncomingWebhook/9f226e3c36fb47249f14d4dab2d5b845/92bac26e-62c5-427f-ac91-e51a268f94ca/V2QaKKUiGE6BMqWd-DeObKqCFmiQE5WSekPuAwjhc6ads1"}, wantErr: false},
		{name: "missing https", args: args{u: "outlook.office.com/webhook/xxxx/xxxx"}, wantErr: true},
		{name: "only http", args: args{u: "http://outlook.office.com/webhook/xxxx/xxxx"}, wantErr: true},
		{name: "https but invalid", args: args{u: "https://example.com"}, wantErr: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if err := validateWebhook(service.O365, tt.args.u); (err != nil) != tt.wantErr {
				t.Errorf("validateWebhook() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_extractWebhookFromRequest(t *testing.T) {
	oldStyleWebhook := "example.webhook.office.com/webhookb2/1e21eb6d-60f4-432f-9428-82cf86ec55a3@b12d4011-2ea0-4377-a99b-35c565546afd/IncomingWebhook/091d20cd2a594db0b2d7ed2937d7bd6d/91f7b1cc-96d0-4612-bedd-8b820c869464"
	newStyleWebhook := "example.webhook.office.com/webhookb2/5c51ab94-86c0-4ba3-a66c-c2ad73acc531@e3ebcd52-aa57-25e8-a214-94fb325450f4/IncomingWebhook/9f226e3c36fb47249f14d4dab2d5b845/92bac26e-62c5-427f-ac91-e51a268f94ca/V2QaKKUiGE6BMqWd-DeObKqCFmiQE5WSekPuAwjhc6ads1"
	tests := []struct {
		name          string
		request       *http.Request
		webhookResult string
		wantErr       bool
	}{
		{
			name:          "standard oldstyle webhook",
			request:       newDummyRequest(fmt.Sprintf("/_dynamicwebhook/%s", oldStyleWebhook), ""),
			webhookResult: fmt.Sprintf("https://%s", oldStyleWebhook),
			wantErr:       false,
		},
		{
			name:          "standard newstyle webhook",
			request:       newDummyRequest(fmt.Sprintf("/_dynamicwebhook/%s", newStyleWebhook), ""),
			webhookResult: fmt.Sprintf("https://%s", newStyleWebhook),
			wantErr:       false,
		},
		{
			name:          "auth header oldstyle webhook",
			request:       newDummyRequest("/_dynamicwebhook/", fmt.Sprintf("webhook %s", oldStyleWebhook)),
			webhookResult: fmt.Sprintf("https://%s", oldStyleWebhook),
			wantErr:       false,
		},
		{
			name:          "auth header newstyle webhook",
			request:       newDummyRequest("/_dynamicwebhook/", fmt.Sprintf("webhook %s", newStyleWebhook)),
			webhookResult: fmt.Sprintf("https://%s", newStyleWebhook),
			wantErr:       false,
		},
		{
			name:    "invalid bearer",
			request: newDummyRequest("/_dynamicwebhook/", fmt.Sprintf("invalid-bearer %s", newStyleWebhook)),
			wantErr: true,
		},
		{
			name:    "missing webhook and header",
			request: newDummyRequest("/_dynamicwebhook/", ""),
			wantErr: true,
		},
	}
	prefix := "/_dynamicwebhook/"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhook, err := extractWebhookFromRequest(tt.request, prefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractWebhookFromRequest() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.webhookResult != webhook {
				t.Errorf("extractWebhookFromRequest() webhook (\"%s\") does not match expected webhook (\"%s\")", webhook, tt.webhookResult)
			}
		})
	}
}

func newDummyRequest(urlPath, authHeader string) *http.Request {
	url := fmt.Sprintf("http://localhost:2000%s", urlPath)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		panic(err)
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	return req
}
