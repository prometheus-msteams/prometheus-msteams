package alert

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

// PrometheusAlertManagerHandler handles incoming request
func (t *Teams) PrometheusAlertManagerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Server only accepts POST requests.", http.StatusMethodNotAllowed)
		return
	}
	if !strings.HasPrefix(t.WebhookURL, "http") {
		msg := fmt.Sprintf("Please check the server webhook configuration for %s\n", r.RequestURI)
		http.Error(w, msg, http.StatusInternalServerError)
		log.Printf("The webhook url for %s is invalid\n", r.RequestURI)
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
	log.Printf("%s received a request from Prometheus Alert Manager\n", r.RequestURI)
	promBytes, _ := json.MarshalIndent(p, " ", "  ")
	fmt.Println(string(promBytes))

	// Create the Card
	// card := new(TeamsMessageCard)
	t.Card.CreateCard(p)

	// For Debugging, display the Request Body to send in JSON Format
	log.Printf("Created a card for Microsoft Teams %s\n", r.RequestURI)
	cardBytes, _ := json.MarshalIndent(t.Card, " ", "  ")
	fmt.Println(string(cardBytes))

	statusCode, err := t.SendCard()
	if err != nil {
		log.Println(err)
		http.Error(w, fmt.Sprintf("%v", err), statusCode)
		return
	}
	CardCounter++
	log.Printf("Total Card sent since uptime: %d\n", CardCounter)
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
