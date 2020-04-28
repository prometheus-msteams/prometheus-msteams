package card

import (
	"context"
	"flag"
	"testing"

	"github.com/prometheus-msteams/prometheus-msteams/pkg/testutils"
)

var update = flag.Bool("update", false, "update .golden files")

func Test_templatedCard_Convert(t *testing.T) {
	tests := []struct {
		name              string
		promAlertFile     string
		templateFile      string
		escapeUnderscores bool
		disableGrouping   bool
		want              map[string]interface{}
		wantErr           bool
	}{
		{
			name:          "do not escape underscores",
			promAlertFile: "./testdata/prom_post_request.json",
			templateFile:  "../../default-message-card.tmpl",
		},
		{
			name:              "escape underscores",
			promAlertFile:     "./testdata/prom_post_request.json",
			templateFile:      "../../default-message-card.tmpl",
			escapeUnderscores: true,
		},
		{
			name:              "disable grouping",
			promAlertFile:     "./testdata/Test_templatedCard_Convert/disable_grouping/prom_post_request.json",
			templateFile:      "../../default-message-card.tmpl",
			escapeUnderscores: true,
			disableGrouping:   true,
		},
		{
			name:              "action card",
			promAlertFile:     "./testdata/prom_post_request.json",
			templateFile:      "./testdata/Test_templatedCard_Convert/action_card/message-card.tmpl",
			escapeUnderscores: true,
		},
		{
			name:              "too many sections",
			promAlertFile:     "./testdata/Test_templatedCard_Convert/too_many_sections/prom_post_request_too_many_sections.json",
			templateFile:      "../../default-message-card.tmpl",
			escapeUnderscores: true,
		},
		{
			name:              "too large",
			promAlertFile:     "./testdata/Test_templatedCard_Convert/too_large/prom_post_request_too_large.json",
			templateFile:      "../../default-message-card.tmpl",
			escapeUnderscores: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := ParseTemplateFile(tt.templateFile)
			if err != nil {
				t.Fatal(err)
			}
			a, err := testutils.ParseWebhookJSONFromFile(tt.promAlertFile)
			if err != nil {
				t.Fatal(err)
			}
			m := NewTemplatedCardCreator(tmpl, tt.escapeUnderscores, tt.disableGrouping)
			got, err := m.Convert(context.Background(), a)
			if (err != nil) != tt.wantErr {
				t.Errorf("templatedCard.Convert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			testutils.CompareToGoldenFile(t, got, t.Name()+"/card.json", *update)
		})
	}
}
