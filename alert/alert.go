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
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// CardCounter displays in the logs
var CardCounter int

// PrometheusAlertMessage is the request body that Prometheus sent via Generic Webhook
// The Documentation is in https://prometheus.io/docs/alerting/configuration/#webhook_config
type PrometheusAlertMessage struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []Alert           `json:"alerts"`
}

// Alert construct is used by the PrometheusAlertMessage.Alerts
type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
	EndsAt      string            `json:"endsAt"`
}

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

// DecodePrometheusAlert decodes Prometheus JSON Request Body to PrometheusAlertMessage struct
func (p *PrometheusAlertMessage) DecodePrometheusAlert(r io.Reader) error {
	decoder := json.NewDecoder(r)
	err := decoder.Decode(p)
	if err != nil {
		return err
	}
	return nil
}

// PrometheusAlertManagerHandler handles incoming request to /alertmanager
func PrometheusAlertManagerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Server only accepts POST requests.", http.StatusMethodNotAllowed)
		return
	}
	var p PrometheusAlertMessage
	err := p.DecodePrometheusAlert(r.Body)
	if err != nil {
		msg := fmt.Sprintf("Server decoding error %v", err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	// For Debugging, display the Request in JSON Format
	log.Println("Request received")
	promBytes, _ := json.MarshalIndent(p, " ", "  ")
	fmt.Println(string(promBytes))

	// Create the Card
	c := new(TeamsMessageCard)
	c.CreateCard(p)

	// For Debugging, display the Request Body to send in JSON Format
	log.Println("Creating a card")
	cardBytes, _ := json.MarshalIndent(c, " ", "  ")
	fmt.Println(string(cardBytes))

	err = c.SendCard()
	if err != nil {
		log.Println(err)
		http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
		return
	}
	CardCounter++
	log.Printf("Total Card sent since uptime: %d\n", CardCounter)
}

// SendCard sends the JSON Encoded TeamsMessageCard
func (c *TeamsMessageCard) SendCard() error {
	buffer := new(bytes.Buffer)
	json.NewEncoder(buffer).Encode(c)
	url := os.Getenv("TEAMS_INCOMING_WEBHOOK_URL")
	res, err := http.Post(url, "application/json", buffer)
	if err != nil {
		if strings.Contains(fmt.Sprintf("%v", err), "Post : unsupported protocol scheme") {
			return fmt.Errorf("%v. The Teams Webhook configuration might be missing or is incorrectly configured in the Server side", err)
		}
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Error: %s", res.Status)
	}
	return nil
}

// CreateCard creates the TeamsMessageCard based on values gathered from PrometheusAlertMessage
func (c *TeamsMessageCard) CreateCard(p PrometheusAlertMessage) error {
	const (
		messageType   = "MessageCard"
		context       = "http://schema.org/extensions"
		colorResolved = "2DC72D"
		colorFiring   = "8C1A1A"
		colorUnknown  = "CCCCCC"
	)
	c.Type = messageType
	c.Context = context
	switch p.Status {
	case "resolved":
		c.ThemeColor = colorResolved
	case "firing":
		c.ThemeColor = colorFiring
	default:
		c.ThemeColor = colorUnknown
	}
	c.Title = fmt.Sprintf("Prometheus Alert (%s)", p.Status)
	if value, notEmpty := p.CommonAnnotations["summary"]; notEmpty {
		c.Summary = value
	}
	useMarkdown := false
	if v := os.Getenv("MARKDOWN_ENABLED"); v == "yes" {
		useMarkdown = true
	}
	for _, alert := range p.Alerts {
		var s TeamsMessageCardSection
		s.ActivityTitle = fmt.Sprintf("[%s](%s)", alert.Annotations["description"], p.ExternalURL)
		s.Markdown = useMarkdown
		for key, val := range alert.Annotations {
			s.Facts = append(s.Facts, TeamsMessageCardSectionFacts{key, val})
		}
		for key, val := range alert.Labels {
			s.Facts = append(s.Facts, TeamsMessageCardSectionFacts{key, val})
		}
		c.Sections = append(c.Sections, s)
	}
	return nil
}
