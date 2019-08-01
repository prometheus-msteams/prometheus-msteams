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
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/template"
	log "github.com/sirupsen/logrus"
)

// Constants for creating a Card
const (
	maxSize         = 14336 // maximum message size of 14336 Bytes (14KB)
	maxCardSections = 10    // maximum number of sections: https://docs.microsoft.com/en-us/microsoftteams/platform/concepts/cards/cards-reference#notes-on-the-office-365-connector-card
)

func counter() func() int {
	i := 0
	return func() int {
		i++
		return i
	}
}

func compact(data []byte) string {
	buffer := new(bytes.Buffer)
	json.Compact(buffer, data)
	return buffer.String()
}

func concatKeyValue(key string, val string) string {
	if strings.HasPrefix(val, "[{") {
		return "\"" + key + "\":" + val
	}
	return "\"" + key + "\":\"" + val + "\""
}

func messageWithoutSections(data []byte) string {
	messageWithoutSections := "{"
	c := counter()
	jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		if string(key) != "sections" {
			if c() != 1 {
				messageWithoutSections += ","
			}
			messageWithoutSections += concatKeyValue(string(key), string(value))
		}
		return nil
	})
	messageWithoutSections += "}"
	return messageWithoutSections
}

func splitTooLargeMessage(data []byte) (string, string) {
	// finalMessage is a valid Teams message card
	finalMessage := "{"
	// restOfMessage is used to recursively apply this method and iteratively create valid Teams message cards
	restOfMessage := "{"

	length := len(messageWithoutSections(data))

	// range over each key-value pair in the original message card
	c1 := counter()
	jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		if string(key) != "sections" {
			if c1() != 1 {
				finalMessage += ","
				restOfMessage += ","
			}
			finalMessage += concatKeyValue(string(key), string(value))
			restOfMessage += concatKeyValue(string(key), string(value))
		}
		if string(key) == "sections" {
			if c1() != 1 {
				finalMessage += ","
				restOfMessage += ","
				length++
			}
			startSections := "\"" + string(key) + "\":["
			finalMessage += startSections
			restOfMessage += startSections
			length++ // for the "]" at the end of the array
			length += len(startSections)
			// counter over section array elements of finalMessage
			c2 := counter()
			// counter over section array elements of restOfMessage
			c3 := counter()
			var counter int
			jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
				section := compact(value)
				length += len(section)
				counter = c2()
				if counter != 1 {
					length++ // for the leading comma sign before appending a new array element
				}
				if (length <= maxSize) && (counter <= maxCardSections) {
					if counter != 1 {
						finalMessage += ","
					}
					finalMessage += section
				} else {
					if c3() != 1 {
						restOfMessage += ","
					}
					restOfMessage += section
				}
			})
			finalMessage += "]"
			restOfMessage += "]"
		}
		return nil
	})
	finalMessage += "}"
	restOfMessage += "}"
	return finalMessage, restOfMessage
}

func querySections(message string) ([]byte, error) {
	sections, _, _, err := jsonparser.Get([]byte(message), "sections")
	return sections, err
}

// CreateCards creates a Teams Message Card based on values gathered from PrometheusWebhook and the structure from the card template
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

	totalMessage, err := webhook.Template.ExecuteTextString(`{{ template "teams.card" . }}`, data)
	if err != nil {
		return "", fmt.Errorf("failed to template alerts: %v", err)
	}
	log.Debugf("Alert rendered in template file: %v", totalMessage)
	totalMessage = compact([]byte(totalMessage))
	var (
		cardTmp          string
		restOfMessageTmp string
	)
	lengthCard := len(totalMessage)
	log.Debugf("Size of message is %d Bytes (~%d KB)", lengthCard, (lengthCard)/(1<<(10*1)))
	cards := "["
	card, restOfMessage := splitTooLargeMessage([]byte(totalMessage))
	cards += card
	missingSections, err := querySections(restOfMessage)
	if err != nil {
		return "", fmt.Errorf("Failed to parse json with key 'sections': %v", err)
	}
	for string(missingSections) != "[]" {
		cardTmp, restOfMessageTmp = splitTooLargeMessage([]byte(restOfMessage))
		cards += "," + cardTmp
		restOfMessage = restOfMessageTmp
		missingSections, err = querySections(restOfMessage)
		if err != nil {
			return "", fmt.Errorf("Failed to parse json with key 'sections': %v", err)
		}
	}
	cards += "]"
	return cards, nil
}

// SendCard sends the Teams message card
func SendCard(webhook string, card string, maxIdleConns int, idleConnTimeout time.Duration, tlsHandshakeTimeout time.Duration) (*http.Response, error) {

	c := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          maxIdleConns,
			IdleConnTimeout:       idleConnTimeout,
			TLSHandshakeTimeout:   tlsHandshakeTimeout,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	req, err := http.NewRequest("POST", webhook, strings.NewReader(card))
	if err != nil {
		return nil, fmt.Errorf("Failed constructing new http request. Got the error: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.Do(req)
	if err != nil {
		return res, fmt.Errorf("Failed sending to webhook url %s. Got the error: %v", webhook, err)
	}

	rb, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error(err)
	}
	log.Infof("Microsoft Teams response text: %s", string(rb))
	if res.StatusCode != http.StatusOK {
		if err != nil {
			return res, fmt.Errorf("Failed reading Teams http response: %v", err)
		}
		return res, fmt.Errorf("Failed sending to the Teams Channel. Teams http response: %s",
			res.Status)
	}
	if err := res.Body.Close(); err != nil {
		log.Error(err)
	}
	return res, nil
}
