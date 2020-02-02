package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prometheus-msteams/prometheus-msteams/pkg/card"
	"github.com/prometheus/alertmanager/notify/webhook"
	"go.opencensus.io/trace"
)

// PostResponse is the prometheus msteams service response.
type PostResponse struct {
	WebhookURL string `json:"webhook_url"`
	Status     int    `json:"status"`
	Message    string `json:"message"`
}

// Service is the Alertmanager to Microsoft Teams webhook service.
type Service interface {
	Post(context.Context, webhook.Message) (resp []PostResponse, err error)
}

type simpleService struct {
	converter  card.Converter
	client     *http.Client
	webhookURL string
}

// NewSimpleService creates a simpleService.
func NewSimpleService(converter card.Converter, client *http.Client, webhookURL string) Service {
	return simpleService{converter, client, webhookURL}
}

func (s simpleService) Post(ctx context.Context, wm webhook.Message) ([]PostResponse, error) {
	ctx, span := trace.StartSpan(ctx, "simpleService.Post")
	defer span.End()

	prs := []PostResponse{}

	jj, err := s.converter.Convert(ctx, wm)
	if err != nil {
		return nil, fmt.Errorf("failed to parse webhook message: %w", err)
	}

	for _, j := range jj {
		pr, err := s.post(ctx, j, s.webhookURL)
		prs = append(prs, pr)
		if err != nil {
			return prs, err
		}
	}

	return prs, nil
}

func (s *simpleService) post(ctx context.Context, j map[string]interface{}, url string) (PostResponse, error) {
	ctx, span := trace.StartSpan(ctx, "simpleService.post")
	defer span.End()

	pr := PostResponse{WebhookURL: url}

	jb, err := json.Marshal(j)
	if err != nil {
		err = fmt.Errorf("failed to decoding JSON card: %w", err)
		return pr, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(jb))
	if err != nil {
		err = fmt.Errorf("failed to creating a request: %w", err)
		return pr, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		err = fmt.Errorf("http client failed: %w", err)
		return pr, err
	}
	defer resp.Body.Close()

	pr.Status = resp.StatusCode

	rb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed reading http response body: %w", err)
		pr.Message = err.Error()
		return pr, err
	}
	pr.Message = string(rb)

	return pr, nil
}
