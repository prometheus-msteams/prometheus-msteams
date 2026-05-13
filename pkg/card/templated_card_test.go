package card

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/prometheus-msteams/prometheus-msteams/pkg/testutils"
)

const (
	testPromAlertFile   = "./testdata/prom_post_request.json"
	testSchemaContext   = "http://schema.org/extensions"
	testAlertTitle      = "Prometheus Alert (Firing)"
	testAlertSummary    = "Prometheus Test"
	testThemeColor      = "FFA500"
	testActivityTitle   = "[10.80.40.11 reported high memory usage with 23.28%.](http://docker.for.mac.host.internal:9093)"
	testMemorySummary   = "Server High Memory usage"
	testAlertname       = "alertname"
	testInstance        = "instance"
	testJob             = "job"
	testMonitor         = "master"
	testSeverity        = "severity"
	testLabelSummary    = "summary"
	testLabelMonitor    = "monitor"
	testSeverityWarning = "warning"
	testTextBlockType   = "TextBlock"
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
			promAlertFile:     testPromAlertFile,
			templateFile:      "../../default-message-card.tmpl",
			escapeUnderscores: false,
			want: Office365ConnectorCard{
				Context:    testSchemaContext,
				Type:       messageCardType,
				Title:      testAlertTitle,
				Summary:    testAlertSummary,
				ThemeColor: testThemeColor,
				Sections: []Section{
					{
						ActivityTitle: testActivityTitle,
						Markdown:      true,
						Facts: []FactSection{
							{},
							{Name: testLabelSummary, Value: testMemorySummary},
							{Name: testAlertname, Value: `high_memory_load`},
							{Name: testInstance, Value: `instance-with-hyphen_and_underscore`},
							{Name: testJob, Value: `docker_nodes`},
							{Name: testLabelMonitor, Value: testMonitor},
							{Name: testSeverity, Value: testSeverityWarning},
						},
					},
				},
			},
		},
		{
			name:              "escape underscores",
			promAlertFile:     testPromAlertFile,
			templateFile:      "../../default-message-card.tmpl",
			escapeUnderscores: true,
			want: Office365ConnectorCard{
				Context:    testSchemaContext,
				Type:       messageCardType,
				Title:      testAlertTitle,
				Summary:    testAlertSummary,
				ThemeColor: testThemeColor,
				Sections: []Section{
					{
						ActivityTitle: testActivityTitle,
						Markdown:      true,
						Facts: []FactSection{
							{},
							{Name: testLabelSummary, Value: testMemorySummary},
							{Name: testAlertname, Value: `high\_memory\_load`},
							{Name: testInstance, Value: `instance-with-hyphen\_and\_underscore`},
							{Name: testJob, Value: `docker\_nodes`},
							{Name: testLabelMonitor, Value: testMonitor},
							{Name: testSeverity, Value: testSeverityWarning},
						},
					},
				},
			},
		},
		{
			name:              "action card",
			promAlertFile:     testPromAlertFile,
			templateFile:      "./testdata/action-message-card.tmpl",
			escapeUnderscores: true,
			want: Office365ConnectorCard{
				Context:    testSchemaContext,
				Type:       messageCardType,
				Title:      testAlertTitle,
				Summary:    testAlertSummary,
				ThemeColor: testThemeColor,
				Sections: []Section{
					{
						ActivityTitle: testActivityTitle,
						Markdown:      true,
						Facts: []FactSection{
							{},
							{Name: testLabelSummary, Value: testMemorySummary},
							{Name: testAlertname, Value: `high\_memory\_load`},
							{Name: testInstance, Value: `instance-with-hyphen\_and\_underscore`},
							{Name: testJob, Value: `docker\_nodes`},
							{Name: testLabelMonitor, Value: testMonitor},
							{Name: testSeverity, Value: testSeverityWarning},
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

func Test_templatedCard_ConvertWorkflow(t *testing.T) {
	tests := []struct {
		name          string
		promAlertFile string
		templateFile  string
		wantErr       bool
		// checkFn allows each case to assert what matters without
		// hard-coding the full card structure.
		checkFn func(t *testing.T, got WorkflowConnectorCard)
	}{
		{
			// The default workflow template uses TextBlock and FactSet elements.
			// Verify that both element types survive the unmarshal/marshal round-trip.
			name:          "default template preserves TextBlock and FactSet",
			promAlertFile: "./testdata/prom_post_request.json",
			templateFile:  "../../default-message-workflow-card.tmpl",
			checkFn: func(t *testing.T, got WorkflowConnectorCard) {
				t.Helper()
				if got.Type != workflowCardType {
					t.Fatalf("want type %q, got %q", workflowCardType, got.Type)
				}
				if len(got.Attachments) != 1 {
					t.Fatalf("want 1 attachment, got %d", len(got.Attachments))
				}
				body := got.Attachments[0].Content.Body
				if len(body) == 0 {
					t.Fatal("body must not be empty")
				}
				var types []string
				for _, elem := range body {
					if typ, ok := elem["type"].(string); ok {
						types = append(types, typ)
					}
				}
				wantTypes := []string{testTextBlockType, testTextBlockType, testTextBlockType, "FactSet"}
				if diff := cmp.Diff(wantTypes, types); diff != "" {
					t.Fatalf("body element types mismatch (-want +got):\n%s", diff)
				}
				// FactSet must carry its facts slice
				factSet := body[len(body)-1]
				facts, ok := factSet["facts"].([]interface{})
				if !ok || len(facts) == 0 {
					t.Fatalf("FactSet element missing facts, got: %v", factSet)
				}
			},
		},
		{
			// A custom template using ColumnSet — an element type not present in
			// the original []Body struct.  Verifies that the looser
			// []map[string]interface{} type preserves all fields (columns, items,
			// style) instead of silently dropping them.
			name:          "custom template preserves ColumnSet",
			promAlertFile: "./testdata/prom_post_request.json",
			templateFile:  "./testdata/workflow_columnset.tmpl",
			checkFn: func(t *testing.T, got WorkflowConnectorCard) {
				t.Helper()
				if got.Type != workflowCardType {
					t.Fatalf("want type %q, got %q", workflowCardType, got.Type)
				}
				if len(got.Attachments) != 1 {
					t.Fatalf("want 1 attachment, got %d", len(got.Attachments))
				}
				body := got.Attachments[0].Content.Body
				if len(body) != 1 {
					t.Fatalf("want 1 body element, got %d", len(body))
				}
				elem := body[0]
				if typ, _ := elem["type"].(string); typ != "ColumnSet" {
					t.Fatalf("want body element type %q, got %q", "ColumnSet", typ)
				}
				if style, _ := elem["style"].(string); style != "attention" {
					t.Fatalf("want ColumnSet style %q, got %q", "attention", style)
				}
				columns, ok := elem["columns"].([]interface{})
				if !ok || len(columns) == 0 {
					t.Fatalf("ColumnSet missing columns, got: %v", elem)
				}
				col, ok := columns[0].(map[string]interface{})
				if !ok {
					t.Fatalf("column is not an object, got: %T", columns[0])
				}
				if colType, _ := col["type"].(string); colType != "Column" {
					t.Fatalf("want column type %q, got %q", "Column", colType)
				}
				items, ok := col["items"].([]interface{})
				if !ok || len(items) == 0 {
					t.Fatalf("Column missing items, got: %v", col)
				}
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

			m := NewTemplatedCardCreator(tmpl, false)

			got, err := m.ConvertWorkflow(context.Background(), a)
			if (err != nil) != tt.wantErr {
				t.Errorf("templatedCard.ConvertWorkflow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkFn != nil {
				tt.checkFn(t, got)
			}
		})
	}
}
