package card

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/alertmanager/notify/webhook"
)

// JSON represents the JSON payload to Microsoft Teams webhook integration.
type JSON []map[string]interface{}

// Converter represents the behavior of a messenger.
type Converter interface {
	// Convert converts an alert manager webhook message to JSON.
	Convert(context.Context, webhook.Message) (JSON, error)
}

type loggingMiddleware struct {
	logger log.Logger
	next   Converter
}

// NewCreatorLoggingMiddleware creates a loggingMiddleware.
func NewCreatorLoggingMiddleware(l log.Logger, n Converter) Converter {
	return loggingMiddleware{l, n}
}

func (l loggingMiddleware) Convert(ctx context.Context, a webhook.Message) (c JSON, err error) {
	defer func(begin time.Time) {
		l.logger.Log(
			"alert", a,
			"card", c,
			"took", time.Since(begin),
		)
	}(time.Now())
	return l.next.Convert(ctx, a)
}
