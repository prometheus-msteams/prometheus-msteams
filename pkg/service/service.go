package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/bzon/prometheus-msteams/pkg/card"
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

	jj, err := s.converter.Convert(wm)
	if err != nil {
		return nil, fmt.Errorf("failed to parse webhook message: %w", err)
	}

	for _, j := range jj {
		pr := PostResponse{WebhookURL: s.webhookURL}
		resp, err := s.post(j)
		if err != nil {
			pr.Message = err.Error()
			prs = append(prs, pr)
			return prs, nil
		}
		pr.Status = resp.StatusCode

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("failed reading http response body: %w", err)
			pr.Message = err.Error()
			prs = append(prs, pr)
			return prs, err
		}
		pr.Message = string(b)

		prs = append(prs, pr)
	}

	return prs, nil
}

func (s *simpleService) post(j map[string]interface{}) (*http.Response, error) {
	b, err := json.Marshal(j)
	if err != nil {
		err = fmt.Errorf("failed to decoding JSON card: %w", err)
		return nil, err
	}

	req, err := http.NewRequest("POST", s.webhookURL, bytes.NewBuffer(b))
	if err != nil {
		err = fmt.Errorf("failed to creating a request: %w", err)
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		err = fmt.Errorf("http client failed: %w", err)
		return nil, err
	}

	return resp, nil
}
