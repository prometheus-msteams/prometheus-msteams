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

	"github.com/buger/jsonparser"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/template"
	log "github.com/sirupsen/logrus"
)

// PrometheusWebhook holds the request uri and the incoming webhook
// Official golang docs (https://golang.org/pkg/net/) contain
// more information on the timeouts
type PrometheusWebhook struct {
	// RequestURI is the request handler for Prometheus to post to
	RequestURI string
	// TeamsWebhookURL is the webhook url of the Teams connector
	TeamsWebhookURL string
	// MarkdownEnabled is used to format the Teams message
	MarkdownEnabled bool
	// template bundles the teams message card based on a template file
	Template *template.Template
	// The maximum number of idle connections allowed
	MaxIdleConns int
	// The idle connection timeout (in seconds)
	IdleConnTimeout int
	// The TLS handshake timeout (in seconds)
	TLSHandshakeTimeout int
}

// String converts the incoming alert to a string
func String(promAlert notify.WebhookMessage) string {
	b, err := json.Marshal(promAlert)
	if err != nil {
		log.Errorf("Failed marshalling Prometheus WebhookMessage: %v", err)
	}
	return string(b)
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
	var promAlert notify.WebhookMessage
	if err := json.NewDecoder(r.Body).Decode(&promAlert); err != nil {
		errMsg := fmt.Sprintf("Failed decoding Prometheus alert message: %v", err)
		log.Error(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	log.Debug(String(promAlert))
	cards, err := CreateCards(promAlert, promWebhook)
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("Created a card for Microsoft Teams %s", r.RequestURI)
	log.Debug(cards)

	jsonparser.ArrayEach([]byte(cards), func(card []byte, dataType jsonparser.ValueType, offset int, err error) {
		cardString := string(card)
		res, err := SendCard(promWebhook.TeamsWebhookURL, cardString, promWebhook.MaxIdleConns, promWebhook.IdleConnTimeout, promWebhook.TLSHandshakeTimeout)
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Infof("A card was successfully sent to Microsoft Teams Channel. Got http status: %s", res.Status)
		if err := res.Body.Close(); err != nil {
			log.Error(err)
		}
	})
}
