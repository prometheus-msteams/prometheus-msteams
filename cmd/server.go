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

package cmd

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/bzon/prometheus-msteams/alert"
	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs the promteams server.",
	Long:  `Runs the promteams server.`,
	Run:   server,
}

var (
	serverPort          int
	serverListenAddress string
	teamsWebhookURL     string
)

func init() {
	RootCmd.AddCommand(serverCmd)
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 2000, "port on which the server will listen")
	serverCmd.Flags().StringVarP(&serverListenAddress, "listen-address", "l", "0.0.0.0", "the address on which the server will listen")
	serverCmd.Flags().StringVarP(&teamsWebhookURL, "webhook-url", "w", "", "the incoming webhook url to post the alert messages")
}

func server(cmd *cobra.Command, args []string) {
	server := serverListenAddress + ":" + strconv.Itoa(serverPort)
	err := os.Setenv("TEAMS_INCOMING_WEBHOOK_URL", teamsWebhookURL)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Printf("promteams server started listening at %s\n", server)
	mux := http.NewServeMux()
	mux.HandleFunc("/alertmanager", alert.PrometheusAlertManagerHandler)
	log.Fatal(http.ListenAndServe(server, mux))
}
