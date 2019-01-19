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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

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

func (promAlert *PrometheusAlertMessage) String() string {
	b, err := json.Marshal(promAlert)
	if err != nil {
		log.Errorf("Failed marshalling PrometheusAlertMessage: %v", err)
	}
	return string(b)
}

// Alert construct is used by the PrometheusAlertMessage.Alerts
type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
	EndsAt      string            `json:"endsAt"`
}

// PrometheusWebhook holds the request uri and the incoming webhook
type PrometheusWebhook struct {
	// RequestURI is the request handler for Prometheus to post to
	RequestURI string
	// TeamsWebhookURL is the webhook url of the Teams connector
	TeamsWebhookURL string
	// MarkdownEnabled is used to format the Teams message
	MarkdownEnabled bool
}

// PrometheusAlertManagerHandler handles incoming request
func (promWebhook *PrometheusWebhook) PrometheusAlertManagerHandler(
	w http.ResponseWriter, r *http.Request) {
	log.Infof("%s received a request", r.RequestURI)
	if r.Method != http.MethodPost {
		errMsg := fmt.Sprintf("Invalid request method: %s, "+
			"this handler only accepts POST requests", r.Method)
		log.Error(errMsg)
		http.Error(w, errMsg, http.StatusMethodNotAllowed)
		return
	}
	if !strings.HasPrefix(promWebhook.TeamsWebhookURL, "http") {
		errMsg := fmt.Sprintf("Invalid webhook url: %s",
			promWebhook.TeamsWebhookURL)
		log.Error(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
	var promAlert PrometheusAlertMessage
	if err := json.NewDecoder(r.Body).Decode(&promAlert); err != nil {
		errMsg := fmt.Sprintf("Failed decoding Prometheus alert message: %v", err)
		log.Error(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	log.Debug(promAlert.String())
	cards := CreateCards(promAlert, promWebhook.MarkdownEnabled)
	log.Infof("Created a card for Microsoft Teams %s", r.RequestURI)
	log.Debug(cards)

	for _, card := range cards {
		res, err := SendCard(promWebhook.TeamsWebhookURL, card)
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Infof("A card was successfully sent to Microsoft Teams Channel. Got http status: %s", res.Status)
		if err := res.Body.Close(); err != nil {
			log.Error(err)
		}
	}
}
