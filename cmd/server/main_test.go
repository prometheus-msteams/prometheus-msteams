package main

import (
	_ "net/http/pprof"
	"testing"
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
		t.Run(tt.name, func(t *testing.T) {
			if err := validateWebhook(tt.args.u); (err != nil) != tt.wantErr {
				t.Errorf("validateWebhook() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
