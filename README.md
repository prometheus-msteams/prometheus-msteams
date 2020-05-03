[![GitHub tag](https://img.shields.io/github/tag/prometheus-msteams/prometheus-msteams.svg)](https://github.com/prometheus-msteams/prometheus-msteams/releases/)
[![Build Status](https://travis-ci.org/prometheus-msteams/prometheus-msteams.svg?branch=master)](https://travis-ci.org/prometheus-msteams/prometheus-msteams)
[![codecov](https://codecov.io/gh/prometheus-msteams/prometheus-msteams/branch/master/graph/badge.svg)](https://codecov.io/gh/prometheus-msteams/prometheus-msteams)
[![Go Report Card](https://goreportcard.com/badge/github.com/prometheus-msteams/prometheus-msteams)](https://goreportcard.com/report/github.com/prometheus-msteams/prometheus-msteams)

![](./docs/teams_screenshot.png)

# Overview

A lightweight Go Web Server that receives __POST__ alert messages from __Prometheus Alert Manager__ and sends it to a __Microsoft Teams Channel__ using an incoming webhook url. How light? See the [docker image](https://quay.io/repository/prometheusmsteams/prometheus-msteams?tab=tags)!

## Synopsis

Alertmanager doesn't support sending to Microsoft Teams out of the box. Fortunately, they allow you to use a generic [webhook_config](https://prometheus.io/docs/alerting/configuration/#webhook_config) for cases like this. This project was inspired from [idealista's](https://github.com/idealista/) [prom2teams](https://github.com/idealista/prom2teams) which was written in Python.

`prometheus-msteams` is a proxy server for Teams. You configure AlertManager `webhook_config` to route the alerts
to prometheus-msteams urls and prometheus-msteams will make the http request to Teams using the webhooks you define.

## Table of Contents

<!-- vim-markdown-toc GFM -->

* [Getting Started](#getting-started)
  * [Creating the configuration file](#creating-the-configuration-file)
  * [Run the server](#run-the-server)
  * [Setting up Prometheus AlertManager](#setting-up-prometheus-alertmanager)
  * [Testing Prometheus Alerts to Teams Channel](#testing-prometheus-alerts-to-teams-channel)
* [Customise Messages to MS Teams](#customise-messages-to-ms-teams)
  * [Custom Message Template per MS Teams Channel](#custom-message-template-per-ms-teams-channel)
* [Configuration](#configuration)
* [Kubernetes](#kubernetes)
* [Contributing](#contributing)

<!-- vim-markdown-toc -->

## Getting Started

![multiChannel](./docs/promteams_multiconfig.png)

### Creating the configuration file

Create a yaml file with with the following. 

```yaml
connectors:
- high_priority_channel: "https://outlook.office.com/webhook/xxxx/aaa/bbb"
- low_priority_channel: "https://outlook.office.com/webhook/xxxx/aaa/ccc"
```

`high_priority_channel` and `low_priority_channel` are arbitrary names for the request paths that prometheus-msteams
will create.

### Run the server

When running as a docker container, mount the config file in the container and set the __CONFIG_FILE__ environment variable.

```bash
docker run -d -p 2000:2000 \
    --name="promteams" \
    -v /tmp/config.yml:/tmp/config.yml \
    -e CONFIG_FILE="/tmp/config.yml" \
    quay.io/prometheusmsteams/prometheus-msteams:v1.4.0
```

When running as a binary, use the __-config-file__ flag.

```bash
./prometheus-msteams server -config-file /tmp/config.yml
```

This will create the request uri handlers __/high_priority_channel__ and __/low_priority_channel__.

To validate your configuration, see the __/config__ endpoint of the application.

```bash
curl localhost:2000/config

[
  {
    "high_priority_channel": "https://outlook.office.com/webhook/xxxx/aaa/bbb"
  },
  {
    "low_priority_channel": "https://outlook.office.com/webhook/xxxx/aaa/ccc"
  }
]
```

### Setting up Prometheus AlertManager

> If you don't have Prometheus running yet and you wan't to try how this works,  
> try [stefanprodan's](https://github.com/stefanprodan) [Prometheus in Docker](https://github.com/stefanprodan/dockprom) to help you install a local Prometheus setup quickly in a single machine.

Now, your AlertManager must have a configuration like the this:

```yaml
route:
  group_by: ['alertname']
  group_interval: 30s
  repeat_interval: 30s
  group_wait: 30s
  receiver: 'low_priority_receiver'  # default/fallback request handler
  routes:
    - receiver: high_priority_receiver
      match:
        severity: critical
    - receiver: low_priority_receiver
      match:
        severity: warning

receivers:
- name: 'high_priority_receiver'
  webhook_configs:
    - send_resolved: true
      url: 'http://localhost:2000/high_priority_channel' # request handler 1
- name: 'low_priority_receiver'
  webhook_configs:
    - send_resolved: true
      url: 'http://localhost:2000/low_priority_channel' # request handler 2
```
Your Prometheus alert could be something like this:

```yaml
alert: SomeAlert
expr: absent(up{job="foo"} == 1)
for: 15m
labels:
  severity: critical # matches the high_priority_channel route.
```

### Testing Prometheus Alerts to Teams Channel

This is just a simple way to test alerts without involving Prometheus AlertManager.

Create the following AlertManager JSON data as `prom-alert.json` and post it to prometheus-msteams server directly.

```json
{
    "version": "4",
    "groupKey": "{}:{alertname=\"high_memory_load\"}",
    "status": "firing",
    "receiver": "teams_proxy",
    "groupLabels": {
        "alertname": "high_memory_load"
    },
    "commonLabels": {
        "alertname": "high_memory_load",
        "monitor": "master",
        "severity": "warning"
    },
    "commonAnnotations": {
        "summary": "Server High Memory usage"
    },
    "externalURL": "http://docker.for.mac.host.internal:9093",
    "alerts": [
        {
            "labels": {
                "alertname": "high_memory_load",
                "instance": "10.80.40.11:9100",
                "job": "docker_nodes",
                "monitor": "master",
                "severity": "warning"
            },
            "annotations": {
                "description": "10.80.40.11 reported high memory usage with 23.28%.",
                "summary": "Server High Memory usage"
            },
            "startsAt": "2018-03-07T06:33:21.873077559-05:00",
            "endsAt": "0001-01-01T00:00:00Z"
        }
    ]
}
```

```bash
curl -X POST -d @prom-alert.json http://localhost:2000/low_priority_channel
```

The teams channel should received a message.

## Customise Messages to MS Teams

This application uses a [default Microsoft Teams Message card template](./default-message-card.tmpl) to convert incoming Prometheus alerts to teams message cards. This template can be customised. Simply create a new file that you want to use as your custom template. It uses the [Go Templating Engine](https://golang.org/pkg/text/template/) and especially the [Prometheus Alertmanager Notification Template](https://prometheus.io/docs/alerting/notifications/). Also see the [Office 365 Connector Card Reference](https://docs.microsoft.com/en-us/microsoftteams/platform/concepts/cards/cards-reference#office-365-connector-card) and some [examples](./examples) for more information to construct your template. Apart from that, you can use the [Message Card Playground](https://messagecardplayground.azurewebsites.net/) to form the basic structure of your card.

When running as a docker container, mount the template file in the container and set the __TEMPLATE_FILE__ environment variable.

```bash
docker run -d -p 2000:2000 \
    --name="promteams" \
    -e TEAMS_INCOMING_WEBHOOK_URL="https://outlook.office.com/webhook/xxx" \
    -v /tmp/card.tmpl:/tmp/card.tmpl \
    -e TEMPLATE_FILE="/tmp/card.tmpl" \
    quay.io/prometheusmsteams/prometheus-msteams
```

When running as a binary, use the __-template-file__ flag.

```bash
./prometheus-msteams server \
    -l localhost \
    -p 2000 \
    -template-file /tmp/card.tmpl
```

### Custom Message Template per MS Teams Channel

You can also use a custom template per webhook by using the `connectors_with_custom_templates`.

```yaml
# alerts in the connectors here will use the default template.
connectors:
- alert1: <webhook> 

# alerts in the connectors here will use template_file specified.
connectors_with_custom_templates:
- request_path: /alert2
  template_file: ./default-message-card.tmpl
  webhook_url: <webhook> 
  escape_underscores: true # get the effect of -auto-escape-underscores.
```

## Configuration

All configuration from flags can be overwritten using environment variables.

E.g, `-config-file` is `CONFIG_FILE`, `-debug` is `DEBUG`, `-log-format` is `LOG_FORMAT`.

```
Usage of prometheus-msteams:
  -auto-escape-underscores
    	Automatically replace all '_' with '\_' from texts in the alert.
  -config-file string
    	The connectors configuration file.
  -debug
    	Set log level to debug mode. (default true)
  -http-addr string
    	HTTP listen address. (default ":2000")
  -idle-conn-timeout duration
    	The HTTP client idle connection timeout duration. (default 1m30s)
  -jaeger-agent string
    	Jaeger agent endpoint (default "localhost:6831")
  -jaeger-trace
    	Send traces to Jaeger.
  -log-format string
    	json|fmt (default "json")
  -max-idle-conns int
    	The HTTP client maximum number of idle connections (default 100)
  -teams-incoming-webhook-url string
    	The default Microsoft Teams webhook connector.
  -teams-request-uri string
    	The default request URI path where Prometheus will post to.
  -template-file string
    	The Microsoft Teams Message Card template file. (default "./default-message-card.tmpl")
  -tls-handshake-timeout duration
    	The HTTP client TLS handshake timeout. (default 30s)
```

## Kubernetes

See [Helm Guide](./chart/prometheus-msteams/README.md).

## Contributing

See [Contributing Guide](./CONTRIBUTING.md)
