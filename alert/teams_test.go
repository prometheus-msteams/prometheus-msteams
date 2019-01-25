package alert

import (
	"encoding/json"
	"io/ioutil"
	"testing"
)

func createTestCards(testdata string, t *testing.T) []*TeamsMessageCard {
	var p PrometheusAlertMessage
	b, err := ioutil.ReadFile(testdata)
	if err != nil {
		t.Fatalf("Failed reading file %s got error: +%v", testdata, err)
	}
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatalf("Failed unmarshalling testdata file %s, got error: +%v",
			testdata, err)
	}
	return CreateCards(p, true)
}

func TestCreateCards(t *testing.T) {
	testdata := "testdata/prom_post_request.json"
	cards := createTestCards(testdata, t)

	if len(cards) != 1 {
		t.Fatalf("TeamsMessageCard.CreatedCard error: should create 1 card, got %d cards", len(cards))
	}

	for _, c := range cards {
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

func TestLargePostRequest(t *testing.T) {
	testdata := "testdata/large_prom_post_request.json"
	cards := createTestCards(testdata, t)

	if len(cards) != 2 {
		t.Fatalf("TeamsMessageCard.CreatedCard error: should create 2 cards, got %d cards", len(cards))
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
		for _, c := range cards {
			if c.ThemeColor != tc.wantColor {
				t.Fatalf("Failed assigning themes color to card: got %s, want %s",
					c.ThemeColor, tc.wantColor)
			}
		}
	}
}
