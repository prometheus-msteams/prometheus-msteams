package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// Constants for Sending a Card
const (
	messageType   = "MessageCard"
	context       = "http://schema.org/extensions"
	colorResolved = "2DC72D"
	colorFiring   = "8C1A1A"
	colorUnknown  = "CCCCCC"
)

// TeamsMessageCard is for the Card Fields to send in Teams
// The Documentation is in https://docs.microsoft.com/en-us/outlook/actionable-messages/card-reference#card-fields
type TeamsMessageCard struct {
	Type       string                    `json:"@type"`
	Context    string                    `json:"@context"`
	ThemeColor string                    `json:"themeColor"`
	Summary    string                    `json:"summary"`
	Title      string                    `json:"title"`
	Text       string                    `json:"text,omitempty"`
	Sections   []TeamsMessageCardSection `json:"sections"`
}

// TeamsMessageCardSection is placed under TeamsMessageCard.Sections
// Each element of AlertWebHook.Alerts will the number of elements of TeamsMessageCard.Sections to create
type TeamsMessageCardSection struct {
	ActivityTitle string                         `json:"activityTitle"`
	Facts         []TeamsMessageCardSectionFacts `json:"facts"`
	Markdown      bool                           `json:"markdown"`
}

// TeamsMessageCardSectionFacts is placed under TeamsMessageCardSection.Facts
type TeamsMessageCardSectionFacts struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Teams holds the request uri and the incoming webhook
type Teams struct {
	RequestURI string
	WebhookURL string
	Card       TeamsMessageCard
}

// SendCard sends the JSON Encoded TeamsMessageCard
func (t *Teams) SendCard() (*http.Response, error) {
	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(t.Card)
	if err != nil {
		return nil, err
	}
	res, err := http.Post(t.WebhookURL, "application/json", buffer)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error: %s", res.Status)
	}
	return res, nil
}

// CreateCard creates the TeamsMessageCard based on values gathered from PrometheusAlertMessage
func (c *TeamsMessageCard) CreateCard(p PrometheusAlertMessage) {
	c.Type = messageType
	c.Context = context
	c.Sections = []TeamsMessageCardSection{}
	switch p.Status {
	case "resolved":
		c.ThemeColor = colorResolved
	case "firing":
		c.ThemeColor = colorFiring
	default:
		c.ThemeColor = colorUnknown
	}
	c.Title = fmt.Sprintf("Prometheus Alert (%s)", p.Status)
	if value, notEmpty := p.CommonAnnotations["summary"]; notEmpty {
		c.Summary = value
	}
	useMarkdown := false
	if v := os.Getenv("MARKDOWN_ENABLED"); v == "yes" {
		useMarkdown = true
	}
	for _, alert := range p.Alerts {
		var s TeamsMessageCardSection
		s.ActivityTitle = fmt.Sprintf("[%s](%s)", alert.Annotations["description"], p.ExternalURL)
		s.Markdown = useMarkdown
		for key, val := range alert.Annotations {
			s.Facts = append(s.Facts, TeamsMessageCardSectionFacts{key, val})
		}
		for key, val := range alert.Labels {
			// Auto escape underscores if markdown is enabled
			if useMarkdown {
				if strings.Contains(val, "_") {
					val = strings.Replace(val, "_", "\\_", -1)
				}
			}
			s.Facts = append(s.Facts, TeamsMessageCardSectionFacts{key, val})
		}
		c.Sections = append(c.Sections, s)
	}
}
