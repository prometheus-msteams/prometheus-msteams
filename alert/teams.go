// Copyright © 2018 bzon <bryansazon@hotmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/template"
	log "github.com/sirupsen/logrus"
)

// Constants for Sending a Card
const (
	messageType     = "MessageCard"
	context         = "http://schema.org/extensions"
	colorResolved   = "2DC72D"
	colorFiring     = "8C1A1A"
	colorUnknown    = "CCCCCC"
	maxSize         = 14336 // maximum message size of 14336 Bytes (14KB)
	maxCardSections = 10    // maximum number of sections: https://docs.microsoft.com/en-us/microsoftteams/platform/concepts/cards/cards-reference#notes-on-the-office-365-connector-card
)

// TeamsMessageCard is for the Card Fields to send in Teams
// The Documentation is in https://docs.microsoft.com/en-us/outlook/actionable-messages/card-reference#card-fields
type TeamsMessageCard struct {
	Type       string                    `json:"@type"`
	Context    string                    `json:"@context"`
	ThemeColor string                    `json:"themeColor"`
	Summary    string                    `json:"summary"`
	Title      string                    `json:"title"`
	Text       string                    `json:"text,omitempty"`
	Sections   []TeamsMessageCardSection `json:"sections"`
}

func (card *TeamsMessageCard) String() string {
	b, err := json.Marshal(card)
	if err != nil {
		log.Errorf("failed marshalling TeamsMessageCard: %v", err)
	}
	return string(b)
}

// TeamsMessageCardSection is placed under TeamsMessageCard.Sections
// Each element of AlertWebHook.Alerts will the number of elements of TeamsMessageCard.Sections to create
type TeamsMessageCardSection struct {
	ActivityTitle string                         `json:"activityTitle"`
	Facts         []TeamsMessageCardSectionFacts `json:"facts"`
	Markdown      bool                           `json:"markdown"`
}

func (section *TeamsMessageCardSection) String() string {
	b, err := json.Marshal(section)
	if err != nil {
		log.Errorf("failed marshalling TeamsMessageCardSection: %v", err)
	}
	return string(b)
}

// TeamsMessageCardSectionFacts is placed under TeamsMessageCardSection.Facts
type TeamsMessageCardSectionFacts struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SendCard sends the JSON Encoded TeamsMessageCard
func SendCard(webhook string, card string) (*http.Response, error) {
	buffer := new(bytes.Buffer)
	if err := json.Compact(buffer, []byte(card)); err != nil {
		return nil, fmt.Errorf("Failed encoding message card: %v", err)
	}
	res, err := http.Post(webhook, "application/json", buffer)
	if err != nil {
		return nil, fmt.Errorf("Failed sending to webhook url %s. Got the error: %v",
			webhook, err)
	}
	rb, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error(err)
	}
	log.Infof("Microsoft Teams response text: %s", string(rb))
	if res.StatusCode != http.StatusOK {
		if err != nil {
			return nil, fmt.Errorf("Failed reading Teams http response: %v", err)
		}
		return nil, fmt.Errorf("Failed sending to the Teams Channel. Teams http response: %s",
			res.Status)
	}
	if err := res.Body.Close(); err != nil {
		log.Error(err)
	}
	return res, nil
}

// CreateCards creates the TeamsMessageCard based on values gathered from PrometheusAlertMessage
func CreateCards(promAlert notify.WebhookMessage, webhook *PrometheusWebhook) (string, error) {

	data := &template.Data{
		Receiver:          promAlert.Receiver,
		Status:            promAlert.Status,
		Alerts:            promAlert.Alerts,
		GroupLabels:       promAlert.GroupLabels,
		CommonLabels:      promAlert.CommonLabels,
		CommonAnnotations: promAlert.CommonAnnotations,
		ExternalURL:       promAlert.ExternalURL,
	}

	card, err := webhook.Template.ExecuteTextString(`{{ template "teams.card" . }}`, data)
	if err != nil {
		return "", fmt.Errorf("failed to template alerts: %v", err)
	}

	return card, nil

	// // maximum message size of 14336 Bytes (14KB)
	// const maxSize = 14336
	// cards := []*TeamsMessageCard{}
	// card := createCardMetadata(promAlert, webhook.MarkdownEnabled)
	// cardMetadataJSON := card.String()
	// cardMetadataSize := len(cardMetadataJSON)
	// // append first card to cards
	// cards = append(cards, card)

	// for _, alert := range promAlert.Alerts {
	// 	var s TeamsMessageCardSection
	// 	s.ActivityTitle = fmt.Sprintf("[%s](%s)",
	// 		alert.Annotations["description"], promAlert.ExternalURL)
	// 	s.Markdown = webhook.MarkdownEnabled
	// 	for key, val := range alert.Annotations {
	// 		s.Facts = append(s.Facts, TeamsMessageCardSectionFacts{key, val})
	// 	}
	// 	for key, val := range alert.Labels {
	// 		// Auto escape underscores if markdown is enabled
	// 		if webhook.MarkdownEnabled {
	// 			if strings.Contains(val, "_") {
	// 				val = strings.Replace(val, "_", "\\_", -1)
	// 			}
	// 		}
	// 		s.Facts = append(s.Facts, TeamsMessageCardSectionFacts{key, val})
	// 	}
	// 	currentCardSize := len(card.String())
	// 	newSectionSize := len(s.String())
	// 	newCardSize := cardMetadataSize + currentCardSize + newSectionSize
	// 	// if total Size of message exceeds maximum message size then split it
	// 	if (newCardSize) < maxSize {
	// 		card.Sections = append(card.Sections, s)
	// 	} else {
	// 		card = createCardMetadata(promAlert, webhook.MarkdownEnabled)
	// 		card.Sections = append(card.Sections, s)
	// 		cards = append(cards, card)
	// 	}
	// }
}
