package alert

import (
	"encoding/json"
	"fmt"
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

// PrometheusAlertManagerHandler handles incoming request
func (t *Teams) PrometheusAlertManagerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Server only accepts POST requests.", http.StatusMethodNotAllowed)
		return
	}
	if !strings.HasPrefix(t.WebhookURL, "http") {
		msg := fmt.Sprintf("Please check the server webhook configuration for %s\n", r.RequestURI)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	var p PrometheusAlertMessage
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		msg := fmt.Sprintf("Failed decoding prometheus alert message: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	log.Printf("%s received a request from Prometheus Alert Manager\n", r.RequestURI)

	// For Debugging, display the Request in JSON Format
	if os.Getenv("PROMTEAMS_DEBUG") == "true" {
		promBytes, _ := json.MarshalIndent(p, " ", "  ")
		fmt.Println(string(promBytes))
	}

	// Create the Card
	t.Card.CreateCard(p)
	log.Printf("Created a card for Microsoft Teams %s\n", r.RequestURI)

	// For Debugging, display the Request Body to send in JSON Format
	if os.Getenv("PROMTEAMS_DEBUG") == "true" {
		cardBytes, _ := json.MarshalIndent(t.Card, "", "  ")
		fmt.Println(string(cardBytes))
	}

	res, err := t.SendCard()
	if err != nil {
		log.Println(err)
		http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()
	log.Println(res.Status)
	CardCounter++
	log.Printf("Total Card sent since uptime: %d\n", CardCounter)
}
