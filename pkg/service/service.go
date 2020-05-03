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

	c, err := s.converter.Convert(ctx, wm)
	if err != nil {
		return nil, fmt.Errorf("failed to parse webhook message: %w", err)
	}

	// Split into multiple messages if necessary.
	cc := splitOffice365Card(c)

	// TODO(@bzon): post concurrently.
	for _, c := range cc {
		pr, err := s.post(ctx, c, s.webhookURL)
		prs = append(prs, pr)
		if err != nil {
			return prs, err
		}
	}

	return prs, nil
}

func (s simpleService) post(ctx context.Context, c card.Office365ConnectorCard, url string) (PostResponse, error) {
	ctx, span := trace.StartSpan(ctx, "simpleService.post")
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

// splitOffice365Card splits a single Office365ConnectorCard into multiple Office365ConnectorCard.
// The purpose of doing this is to prevent getting limited by Microsoft Teams API when sending a large JSON payload.
func splitOffice365Card(c card.Office365ConnectorCard) []card.Office365ConnectorCard {
	// Maximum number of sections
	// ref: https://docs.microsoft.com/en-us/microsoftteams/platform/concepts/cards/cards-reference#notes-on-the-office-365-connector-card
	const maxCardSections = 10

	var cards []card.Office365ConnectorCard

	// Everything is good.
	if len(c.Sections) < maxCardSections {
		cards = append(cards, c)
		return cards
	}

	indexAdded := make(map[int]bool)

	// Here, we keep creating a new card until all sections are transferred into a new card.
	for len(indexAdded) != len(c.Sections) {
		newCard := c // take all the attributes
		newCard.Sections = nil

		for i, s := range c.Sections {
			if _, ok := indexAdded[i]; ok { // check if the index is already added.
				continue
			}

			// If the max length or size has exceeded the limit,
			// break the loop so we can create a new card again.
			if len(newCard.Sections) >= maxCardSections {
				break
			}

			newCard.Sections = append(newCard.Sections, s)
			indexAdded[i] = true
		}

		cards = append(cards, newCard)
	}

	return cards
}
