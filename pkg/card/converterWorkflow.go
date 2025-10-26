package card

import (
	"context"
	"time"

	"github.com/prometheus/alertmanager/notify/webhook"
)

type FactSectionWorkflow struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

type Body struct {
	Type   string                `json:"type"`
	Text   string                `json:"text"`
	Weight string                `json:"weight,omitempty"`
	Size   string                `json:"size,omitempty"`
	Wrap   bool                  `json:"wrap,omitempty"`
	Style  string                `json:"style,omitempty"`
	Color  string                `json:"color,omitempty"`
	Bleed  bool                  `json:"bleed,omitempty"`
	Facts  []FactSectionWorkflow `json:"facts,omitempty"`
}

type BackgroundImage struct {
	Url      string `json:"url"`
	FillMode string `json:"fillMode,omitempty"`
}

type Content struct {
	Schema          string          `json:"$schema"`
	Type            string          `json:"type"`
	Version         string          `json:"version"`
	Body            []Body          `json:"body"`
	Actions         []Action        `json:"actions,omitempty"`
	BackgroundImage BackgroundImage `json:"backgroundImage,omitempty"`
}

type AdaptiveCardItem struct {
	ContentType string  `json:"contentType"` // Always "application/vnd.microsoft.card.adaptive"
	ContentURL  *string `json:"contentUrl"`  // Use a pointer to handle null values
	Content     Content `json:"content"`
}

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

		l.logger.Log(
			"alert", a,
			"card", c,
			"took", time.Since(begin),
		)
	}(time.Now())
	return l.next.ConvertWorkflow(ctx, a)
}
