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
	"github.com/spf13/viper"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs the prometheus-mteams server.",
	Long: `
By using a --config file, you will be able to define multiple prometheus request uri and webhook for different channels.

This is an example config file content in YAML format.

---
connectors:
- channel_1: https://outlook.office.com/webhook/xxxx/hook/for/channel1
- channel_2: https://outlook.office.com/webhook/xxxx/hook/for/channel2

`,
	Run: server,
}

var (
	serverPort          int
	serverListenAddress string
	teamsWebhookURL     string
	requestURI          string
)

// TeamsConfig is the struct for config files
type TeamsConfig struct {
	Connectors []map[string]string
}

func init() {
	RootCmd.AddCommand(serverCmd)
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 2000, "port on which the server will listen")
	serverCmd.Flags().StringVarP(&serverListenAddress, "listen-address", "l", "0.0.0.0", "the address on which the server will listen")
	serverCmd.Flags().StringVarP(&requestURI, "request-uri", "r", "alertmanager", "the request uri path. Do not use this if using a config file.")
	serverCmd.Flags().StringVarP(&teamsWebhookURL, "webhook-url", "w", "", "the incoming webhook url to post the alert messages. Do not use this if using a config file.")
}

func server(cmd *cobra.Command, args []string) {
	mux := http.NewServeMux()
	teamsCfg := new(TeamsConfig)
	viper.Unmarshal(teamsCfg)
	if len(teamsCfg.Connectors) == 0 {
		if len(requestURI) == 0 || len(teamsWebhookURL) == 0 {
			log.Println("A config file (-f) or --request-uri or --webhook-url is not found.")
			cmd.Usage()
			os.Exit(1)
		}
		handleMuxFuncs(requestURI, teamsWebhookURL, mux)
	} else {
		for _, teamMap := range teamsCfg.Connectors {
			for uri, webhook := range teamMap {
				handleMuxFuncs(uri, webhook, mux)
			}
		}
	}
	server := serverListenAddress + ":" + strconv.Itoa(serverPort)
	log.Printf("prometheus-msteams server started listening at %s\n", server)
	log.Fatal(http.ListenAndServe(server, mux))
}

func handleMuxFuncs(uri string, webhook string, mux *http.ServeMux) {
	team := alert.Teams{
		RequestURI: "/" + uri,
		WebhookURL: webhook,
	}
	log.Printf("Adding request uri path %s\n", team.RequestURI)
	mux.HandleFunc(team.RequestURI, team.PrometheusAlertManagerHandler)
}
