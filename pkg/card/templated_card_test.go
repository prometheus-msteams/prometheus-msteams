package card

import (
	"context"
	"flag"
	"testing"

	"github.com/bzon/prometheus-msteams/pkg/testutils"
)

var update = flag.Bool("update", false, "update .golden files")

func Test_templatedCard_Convert(t *testing.T) {
	tests := []struct {
		name          string
		promAlertFile string
		templateFile  string
		want          map[string]interface{}
		wantErr       bool
	}{
		{
			name:          "smoke test",
			promAlertFile: "./testdata/prom_post_request.json",
			templateFile:  "../../default-message-card.tmpl",
		},
		// TODO: add negative test for errors.
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
			m := NewTemplatedCardCreator(tmpl)
			got, err := m.Convert(context.Background(), a)
			if (err != nil) != tt.wantErr {
				t.Errorf("templatedCard.Convert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			testutils.CompareToGoldenFile(t, got, t.Name()+"/card.json", *update)
		})
	}
}
