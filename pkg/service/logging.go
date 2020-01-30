package service

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/alertmanager/notify/webhook"
)

// loggingService is a logging middleware for Service.
type loggingService struct {
	logger log.Logger
	next   Service
}

// NewLoggingService creates a loggingService.
func NewLoggingService(logger log.Logger, next Service) Service {
	return loggingService{logger, next}
}

func (s loggingService) Post(ctx context.Context, wm webhook.Message) (prs []PostResponse, err error) {
	defer func() {
		for _, pr := range prs {
			level.Debug(s.logger).Log(
				"response_message", pr.Message,
				"response_status", pr.Status,
				"webhook_url", pr.WebhookURL,
				"err", err,
			)
		}
	}()
	return s.next.Post(ctx, wm)
}
