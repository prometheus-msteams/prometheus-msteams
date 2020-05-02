package service

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/prometheus-msteams/prometheus-msteams/pkg/card"
)

func Test_splitOffice365Card(t *testing.T) {
	tests := []struct {
		name string
		c    card.Office365ConnectorCard
		want []card.Office365ConnectorCard
	}{
		{
			name: "no split required",
			c: card.Office365ConnectorCard{
				Context:  "http://schema.org/extensions",
				Type:     "MessageCard",
				Sections: []card.Section{{ActivityTitle: "1"}},
			},
			want: []card.Office365ConnectorCard{
				{
					Context:  "http://schema.org/extensions",
					Type:     "MessageCard",
					Sections: []card.Section{{ActivityTitle: "1"}},
				},
			},
		},
		{
			name: "too many sections must be splitted",
			c: card.Office365ConnectorCard{
				Context: "http://schema.org/extensions",
				Type:    "MessageCard",
				Sections: []card.Section{
					{ActivityTitle: "1"},
					{ActivityTitle: "2"},
					{ActivityTitle: "3"},
					{ActivityTitle: "4"},
					{ActivityTitle: "5"},
					{ActivityTitle: "6"},
					{ActivityTitle: "7"},
					{ActivityTitle: "8"},
					{ActivityTitle: "9"},
					{ActivityTitle: "10"},
					{ActivityTitle: "11"},
					{ActivityTitle: "12"},
					{ActivityTitle: "13"},
					{ActivityTitle: "14"},
					{ActivityTitle: "15"},
					{ActivityTitle: "16"},
					{ActivityTitle: "17"},
					{ActivityTitle: "18"},
					{ActivityTitle: "19"},
					{ActivityTitle: "20"},
				},
			},
			want: []card.Office365ConnectorCard{
				{
					Context: "http://schema.org/extensions",
					Type:    "MessageCard",
					Sections: []card.Section{
						{ActivityTitle: "1"},
						{ActivityTitle: "2"},
						{ActivityTitle: "3"},
						{ActivityTitle: "4"},
						{ActivityTitle: "5"},
						{ActivityTitle: "6"},
						{ActivityTitle: "7"},
						{ActivityTitle: "8"},
						{ActivityTitle: "9"},
						{ActivityTitle: "10"},
					},
				},
				{
					Context: "http://schema.org/extensions",
					Type:    "MessageCard",
					Sections: []card.Section{
						{ActivityTitle: "11"},
						{ActivityTitle: "12"},
						{ActivityTitle: "13"},
						{ActivityTitle: "14"},
						{ActivityTitle: "15"},
						{ActivityTitle: "16"},
						{ActivityTitle: "17"},
						{ActivityTitle: "18"},
						{ActivityTitle: "19"},
						{ActivityTitle: "20"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := splitOffice365Card(tt.c)

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
