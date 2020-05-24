package card

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/prometheus-msteams/prometheus-msteams/pkg/testutils"
)

func Test_templatedCard_Convert(t *testing.T) {
	tests := []struct {
		name              string
		promAlertFile     string
		templateFile      string
		escapeUnderscores bool
		want              Office365ConnectorCard
		wantErr           bool
	}{
		{
			name:              "do not escape underscores",
			promAlertFile:     "./testdata/prom_post_request.json",
			templateFile:      "../../default-message-card.tmpl",
			escapeUnderscores: false,
			want: Office365ConnectorCard{
				Context:    "http://schema.org/extensions",
				Type:       "MessageCard",
				Title:      "Prometheus Alert (Firing)",
				Summary:    "Prometheus Test",
				ThemeColor: "FFA500",
				Sections: []Section{
					{
						ActivityTitle: "[10.80.40.11 reported high memory usage with 23.28%.](http://docker.for.mac.host.internal:9093)",
						Markdown:      true,
						Facts: []FactSection{
							{
								Name:  "description",
								Value: "10.80.40.11 reported high memory usage with 23.28%.",
							},
							{Name: "summary", Value: "Server High Memory usage"},
							{Name: "alertname", Value: `high_memory_load`},
							{Name: "instance", Value: `instance-with-hyphen_and_underscore`},
							{Name: "job", Value: `docker_nodes`},
							{Name: "monitor", Value: "master"},
							{Name: "severity", Value: "warning"},
						},
					},
				},
			},
		},
		{
			name:              "escape underscores",
			promAlertFile:     "./testdata/prom_post_request.json",
			templateFile:      "../../default-message-card.tmpl",
			escapeUnderscores: true,
			want: Office365ConnectorCard{
				Context:    "http://schema.org/extensions",
				Type:       "MessageCard",
				Title:      "Prometheus Alert (Firing)",
				Summary:    "Prometheus Test",
				ThemeColor: "FFA500",
				Sections: []Section{
					{
						ActivityTitle: "[10.80.40.11 reported high memory usage with 23.28%.](http://docker.for.mac.host.internal:9093)",
						Markdown:      true,
						Facts: []FactSection{
							{
								Name:  "description",
								Value: "10.80.40.11 reported high memory usage with 23.28%.",
							},
							{Name: "summary", Value: "Server High Memory usage"},
							{Name: "alertname", Value: `high\_memory\_load`},
							{Name: "instance", Value: `instance-with-hyphen\_and\_underscore`},
							{Name: "job", Value: `docker\_nodes`},
							{Name: "monitor", Value: "master"},
							{Name: "severity", Value: "warning"},
						},
					},
				},
			},
		},
		{
			name:              "action card",
			promAlertFile:     "./testdata/prom_post_request.json",
			templateFile:      "./testdata/action-message-card.tmpl",
			escapeUnderscores: true,
			want: Office365ConnectorCard{
				Context:    "http://schema.org/extensions",
				Type:       "MessageCard",
				Title:      "Prometheus Alert (Firing)",
				Summary:    "Prometheus Test",
				ThemeColor: "FFA500",
				Sections: []Section{
					{
						ActivityTitle: "[10.80.40.11 reported high memory usage with 23.28%.](http://docker.for.mac.host.internal:9093)",
						Markdown:      true,
						Facts: []FactSection{
							{
								Name:  "description",
								Value: "10.80.40.11 reported high memory usage with 23.28%.",
							},
							{Name: "summary", Value: "Server High Memory usage"},
							{Name: "alertname", Value: `high\_memory\_load`},
							{Name: "instance", Value: `instance-with-hyphen\_and\_underscore`},
							{Name: "job", Value: `docker\_nodes`},
							{Name: "monitor", Value: "master"},
							{Name: "severity", Value: "warning"},
						},
					},
				},
				PotentialAction: []Action{
					{
						"@context": string("http://schema.org"),
						"@type":    string("ViewAction"),
						"name":     string("Runbook"),
						"target":   []interface{}{string("https://github.com/bzon/prometheus-msteams")},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := ParseTemplateFile(tt.templateFile)
			if err != nil {
				t.Fatal(err)
			}

			a, err := testutils.ParseWebhookJSONFromFile(tt.promAlertFile)
			if err != nil {
				t.Fatal(err)
			}

			m := NewTemplatedCardCreator(tmpl, tt.escapeUnderscores)

			got, err := m.Convert(context.Background(), a)
			if (err != nil) != tt.wantErr {
				t.Errorf("templatedCard.Convert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
