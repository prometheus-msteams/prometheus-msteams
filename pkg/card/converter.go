package card

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/alertmanager/notify/webhook"
)

// Office365ConnectorCard represents https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference#example-office-365-connector-card
type Office365ConnectorCard struct {
	Context         string    `json:"@context"`
	Type            string    `json:"@type"`
	Title           string    `json:"title"`
	ExpectedActors  []string  `json:"expectedActors,omitempty"`
	Text            string    `json:"text"`
	Summary         string    `json:"summary"`
	ThemeColor      string    `json:"themeColor"`
	Sections        []Section `json:"sections,omitempty"`
	PotentialAction []Action  `json:"potentialAction,omitempty"`
}

// Image represents https://docs.microsoft.com/en-us/outlook/actionable-messages/message-card-reference#image-object
type Image struct {
	Image string `json:"image,omitempty"`
	Title string `json:"title,omitempty"`
}

// Action represents https://docs.microsoft.com/en-us/outlook/actionable-messages/message-card-reference#actions
// It is currently impossible to support each type in one struct. This is Go's limitation.
type Action map[string]interface{}

// Section represents https://docs.microsoft.com/en-us/outlook/actionable-messages/message-card-reference#section-fields
type Section struct {
	Title            string        `json:"title,omitempty"`
	ActivityTitle    string        `json:"activityTitle,omitempty"`
	ActivityText     string        `json:"activityText,omitempty"`
	ActivitySubtitle string        `json:"activitySubtitle,omitempty"`
	ActivityImage    string        `json:"activityImage,omitempty"`
	Text             string        `json:"text,omitempty"`
	Markdown         bool          `json:"markdown"`
	Facts            []FactSection `json:"facts,omitempty"`
	Images           []Image       `json:"images,omitempty"`
	PotentialAction  []Action      `json:"potentialAction,omitempty"`
}

type FactSection struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Converter converts an alert manager webhook message to Office365ConnectorCard.
type Converter interface {
	Convert(context.Context, webhook.Message) (Office365ConnectorCard, error)
}

type loggingMiddleware struct {
	logger log.Logger
	next   Converter
}

// NewCreatorLoggingMiddleware creates a loggingMiddleware.
func NewCreatorLoggingMiddleware(l log.Logger, n Converter) Converter {
	return loggingMiddleware{l, n}
}

func (l loggingMiddleware) Convert(ctx context.Context, a webhook.Message) (c Office365ConnectorCard, err error) {
	defer func(begin time.Time) {
		if len(c.PotentialAction) > 5 {
			l.logger.Log(
				"warning", "There can only be a maximum of 5 actions in a potentialAction collection",
				"actions", c.PotentialAction,
			)
		}

		for _, s := range c.Sections {
			if len(s.PotentialAction) > 5 {
				l.logger.Log(
					"warning", "There can only be a maximum of 5 actions in a potentialAction collection",
					"actions", s.PotentialAction,
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
