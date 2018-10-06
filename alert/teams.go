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
	"strings"

	log "github.com/sirupsen/logrus"
)

// Constants for Sending a Card
const (
	messageType   = "MessageCard"
	context       = "http://schema.org/extensions"
	colorResolved = "2DC72D"
	colorFiring   = "8C1A1A"
	colorUnknown  = "CCCCCC"
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

// TeamsMessageCardSection is placed under TeamsMessageCard.Sections
// Each element of AlertWebHook.Alerts will the number of elements of TeamsMessageCard.Sections to create
type TeamsMessageCardSection struct {
	ActivityTitle string                         `json:"activityTitle"`
	Facts         []TeamsMessageCardSectionFacts `json:"facts"`
	Markdown      bool                           `json:"markdown"`
}

// TeamsMessageCardSectionFacts is placed under TeamsMessageCardSection.Facts
type TeamsMessageCardSectionFacts struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SendCard sends the JSON Encoded TeamsMessageCard
func SendCard(webhook string, c *TeamsMessageCard) (*http.Response, error) {
	buffer := new(bytes.Buffer)
	if err := json.NewEncoder(buffer).Encode(c); err != nil {
		return nil, fmt.Errorf("Failed encoding message card: %v", err)
	}
	res, err := http.Post(webhook, "application/json", buffer)
	if err != nil {
		return nil, fmt.Errorf("Failed sending to webhook url %s. Got the error: %v",
			webhook, err)
	}
	if res.StatusCode != http.StatusOK {
		resMessage, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("Failed reading Teams http response: %v", err)
		}
		return nil, fmt.Errorf("Failed sending to the Teams Channel. Teams http response: %s, %s",
			res.Status, string(resMessage))
	}
	if err := res.Body.Close(); err != nil {
		log.Error(err)
	}
	return res, nil
}

// CreateCard creates the TeamsMessageCard based on values gathered from PrometheusAlertMessage
func CreateCard(p PrometheusAlertMessage, markdownEnabled bool) *TeamsMessageCard {
	c := &TeamsMessageCard{}
	c.Type = messageType
	c.Context = context
	c.Sections = []TeamsMessageCardSection{}
	switch p.Status {
	case "resolved":
		c.ThemeColor = colorResolved
	case "firing":
		c.ThemeColor = colorFiring
	default:
		c.ThemeColor = colorUnknown
	}
	c.Title = fmt.Sprintf("Prometheus Alert (%s)", p.Status)
	// Set a default Summary, this is required for Microsoft Teams
	c.Summary = "Prometheus Alert received"
	// Override the value of the Summary if the common annotation exists
	if value, ok := p.CommonAnnotations["summary"]; ok {
		c.Summary = value
	}
	for _, alert := range p.Alerts {
		var s TeamsMessageCardSection
		s.ActivityTitle = fmt.Sprintf("[%s](%s)",
			alert.Annotations["description"], p.ExternalURL)
		s.Markdown = markdownEnabled
		for key, val := range alert.Annotations {
			s.Facts = append(s.Facts, TeamsMessageCardSectionFacts{key, val})
		}
		for key, val := range alert.Labels {
			// Auto escape underscores if markdown is enabled
			if markdownEnabled {
				if strings.Contains(val, "_") {
					val = strings.Replace(val, "_", "\\_", -1)
				}
			}
			s.Facts = append(s.Facts, TeamsMessageCardSectionFacts{key, val})
		}
		c.Sections = append(c.Sections, s)
	}
	return c
}

func (c *TeamsMessageCard) String() string {
	b, err := json.Marshal(c)
	if err != nil {
		log.Errorf("failed marshalling TeamsMessageCard: %v", err)
	}
	return string(b)
}
