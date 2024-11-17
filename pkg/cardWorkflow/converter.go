package cardWorkflow

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus-msteams/prometheus-msteams/pkg/card"
	"github.com/prometheus/alertmanager/notify/webhook"
)

type FactSection struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

type Body struct {
	Type   string        `json:"type"`
	Text   string        `json:"text"`
	Weight string        `json:"weigth,omitempty"`
	Size   string        `json:"size,omitempty"`
	Wrap   bool          `json:"wrap,omitempty"`
	Style  string        `json:"style,omitempty"`
	Color  string        `json:"color,omitempty"`
	Bleed  bool          `json:"bleed,omitempty"`
	Facts  []FactSection `json:"facts,omitempty"`
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
	Actions         []card.Action   `json:"actions,omitempty"`
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

// Converter converts an alert manager webhook message to WorkflowConnectorCard.
type Converter interface {
	Convert(context.Context, webhook.Message) (WorkflowConnectorCard, error)
}

type loggingMiddleware struct {
	logger log.Logger
	next   Converter
}

// NewCreatorLoggingMiddleware creates a loggingMiddleware.
func NewCreatorLoggingMiddleware(l log.Logger, n Converter) Converter {
	return loggingMiddleware{l, n}
}

func (l loggingMiddleware) Convert(ctx context.Context, a webhook.Message) (c WorkflowConnectorCard, err error) {
	defer func(begin time.Time) {
		// if len(c.Actions) > 5 {
		// 	l.logger.Log(
		// 		"warning", "There can only be a maximum of 5 actions in a potentialAction collection",
		// 		"actions", c.Actions,
		// 	)
		// }

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
	return l.next.Convert(ctx, a)
}
