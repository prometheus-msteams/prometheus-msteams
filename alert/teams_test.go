package alert

import (
	"encoding/json"
	"io/ioutil"
	"testing"
)

func TestCreateCards(t *testing.T) {
	var p PrometheusAlertMessage
	testdata := "testdata/prom_post_request.json"
	b, err := ioutil.ReadFile(testdata)
	if err != nil {
		t.Fatalf("Failed reading file %s got error: +%v", testdata, err)
	}
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatalf("Failed unmarshalling testdata file %s, got error: +%v",
			testdata, err)
	}
	cards := CreateCards(p, true)

	for _,c := range cards {
		want := colorFiring
		got := c.ThemeColor
		if got != want {
			t.Fatalf("TeamsMessageCard.CreatedCard error: got %s, want %s", got, want)
		}
	
		want = "Server High Memory usage"
		got = c.Summary
		if got != want {
			t.Fatalf("TeamsMessageCard.CreatedCard error: got %s, want %s", got, want)
		}
	}
}

func TestStatusColor(t *testing.T) {
	tt := []struct {
		status    string
		wantColor string
	}{
		{"firing", colorFiring},
		{"resolved", colorResolved},
		{"unknown", colorUnknown},
	}

	for _, tc := range tt {
		p := PrometheusAlertMessage{Status: tc.status}
		cards := CreateCards(p, true)
		for _,c := range cards {
			if c.ThemeColor != tc.wantColor {
				t.Fatalf("Failed assigning themes color to card: got %s, want %s",
					c.ThemeColor, tc.wantColor)
			}
		}
	}
}
