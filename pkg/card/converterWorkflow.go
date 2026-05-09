package card

import (
	"context"
	"time"

	"github.com/prometheus/alertmanager/notify/webhook"
)

// BackgroundImage represents the background image configuration for an adaptive card.
type BackgroundImage struct {
	URL      string `json:"url"`
	FillMode string `json:"fillMode,omitempty"`
}

// MsTeams represents Microsoft Teams-specific configuration in an adaptive card.
type MsTeams struct {
	Width string `json:"width"`
}

// Content represents the content of an adaptive card for Workflow connector cards.
type Content struct {
	Schema          string                   `json:"$schema"`
	Type            string                   `json:"type"`
	Version         string                   `json:"version"`
	Body            []map[string]interface{} `json:"body"`
	MsTeams         MsTeams                  `json:"msteams"`
	Actions         []Action                 `json:"actions,omitempty"`
	BackgroundImage BackgroundImage          `json:"backgroundImage,omitempty"`
}

// AdaptiveCardItem represents an adaptive card item within a Workflow connector card attachment.
type AdaptiveCardItem struct {
	ContentType string  `json:"contentType"` // Always "application/vnd.microsoft.card.adaptive"
	ContentURL  *string `json:"contentUrl"`  // Use a pointer to handle null values
	Content     Content `json:"content"`
}

// WorkflowConnectorCard represents a Microsoft Teams Workflow connector card message.
type WorkflowConnectorCard struct {
	Type        string             `json:"type"`
	Attachments []AdaptiveCardItem `json:"attachments"`
}

func (l loggingMiddleware) ConvertWorkflow(ctx context.Context, a webhook.Message) (c WorkflowConnectorCard, err error) {
	defer func(begin time.Time) {
		for _, attachment := range c.Attachments {
			if len(attachment.Content.Actions) > 5 {
				l.logger.Log(
					"warning", "There can only be a maximum of 5 actions in a action collection",
					"actions", attachment.Content.Actions,
				)
			}
		}

		_ = l.logger.Log(
			"alert", a,
			"card", c,
			"took", time.Since(begin),
		)
	}(time.Now())
	return l.next.ConvertWorkflow(ctx, a)
}
