package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prometheus-msteams/prometheus-msteams/pkg/cardWorkflow"
	"github.com/prometheus/alertmanager/notify/webhook"
	"go.opencensus.io/trace"
)

type workflowService struct {
	converter  cardWorkflow.Converter
	client     *http.Client
	webhookURL string
}

// NewWorkflowService creates a workflowService.
func NewWorkflowService(converter cardWorkflow.Converter, client *http.Client, webhookURL string) Service {
	return workflowService{converter, client, webhookURL}
}

func (s workflowService) Post(ctx context.Context, wm webhook.Message) ([]PostResponse, error) {
	ctx, span := trace.StartSpan(ctx, "workflowService.Post")
	defer span.End()

	prs := []PostResponse{}

	c, err := s.converter.Convert(ctx, wm)
	if err != nil {
		return nil, fmt.Errorf("failed to parse webhook message: %w", err)
	}

	// TODO(@bzon): post concurrently.
	_, err = s.post(ctx, c, s.webhookURL)
	if err != nil {
		return prs, err
	}

	return prs, nil
}

func (s workflowService) post(ctx context.Context, c cardWorkflow.WorkflowConnectorCard, url string) (PostResponse, error) {
	ctx, span := trace.StartSpan(ctx, "workflowService.post")
	defer span.End()

	pr := PostResponse{WebhookURL: url}

	b, err := json.Marshal(c)
	if err != nil {
		err = fmt.Errorf("failed to decoding JSON card: %w", err)
		return pr, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(b))
	if err != nil {
		err = fmt.Errorf("failed to creating a request: %w", err)
		return pr, err
	}

	req.Header.Set("Content-Type", "application/json")
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
