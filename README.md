![Docker Pulls](https://img.shields.io/docker/pulls/bzon/prometheus-msteams.svg)
[![codecov](https://codecov.io/gh/bzon/prometheus-msteams/branch/master/graph/badge.svg)](https://codecov.io/gh/bzon/prometheus-msteams)
[![Go Report Card](https://goreportcard.com/badge/github.com/bzon/prometheus-msteams)](https://goreportcard.com/report/github.com/bzon/prometheus-msteams)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/bzon/prometheus-msteams)
[![GitHub tag](https://img.shields.io/github/tag/bzon/prometheus-msteams.svg)](https://github.com/bzon/prometheus-msteams/releases/)
[![CircleCI](https://circleci.com/gh/bzon/prometheus-msteams.svg?style=svg)](https://circleci.com/gh/bzon/prometheus-msteams)

![](./docs/teams_screenshot.png)

# Overview

A lightweight Go Web Server that receives __POST__ alert messages from __Prometheus Alert Manager__ and sends it to a __Microsoft Teams Channel__ using an incoming webhook url. How light? The [docker image](https://hub.docker.com/r/bzon/prometheus-msteams/tags/) is just __7 MB__!

## Why choose Go? Not Python or Ruby or Node?

Why use [Go](https://golang.org/)? A Go binary is statically compiled unlike the other simple language (python, ruby, node). Having a static binary means that there is no need for you to install your program's dependencies and these dependencies takes up a lot of space in your docker image! Try it out DevOps folks!

## Table of Contents

<!-- vim-markdown-toc GFM -->

* [Getting Started (Quickstart)](#getting-started-quickstart)
	* [Installation](#installation)
	* [Setting up Prometheus Alert Manager](#setting-up-prometheus-alert-manager)
	* [Simulating a Prometheus Alerts to Teams Channel](#simulating-a-prometheus-alerts-to-teams-channel)
* [Sending Alerts to Multiple Teams Channel](#sending-alerts-to-multiple-teams-channel)
	* [Creating the Configuration File](#creating-the-configuration-file)
	* [Setting up Prometheus Alert Manager](#setting-up-prometheus-alert-manager-1)
* [Debugging](#debugging)
* [Development and Testing](#development-and-testing)
* [Why?](#why)

<!-- vim-markdown-toc -->

## Getting Started (Quickstart)

How it works.

![](./docs/promteams.png)

### Installation

__OPTION 1:__ Run using docker.

```bash
docker run -d -p 2000:2000 \
    --name="promteams" \
    -e TEAMS_INCOMING_WEBHOOK_URL="https://outlook.office.com/webhook/xxx" \
    -e TEAMS_REQUEST_URI=alertmanager \
    -e PROMTEAMS_DEBUG="true" \
    docker.io/bzon/prometheus-msteams:latest
```

__OPTION 2:__ Run using binary.

Download the binary for your platform from [RELEASES](https://github.com/bzon/prometheus-msteams/releases), and run it like the following:

```bash
PROMTEAMS_DEBUG="true" ./prometheus-msteams server \
	-l localhost \
	-p 2000 \
	-w "https://outlook.office.com/webhook/xxx"
```

### Setting up Prometheus Alert Manager

By default, __prometheus-msteams__ creates a request uri handler __/alertmanager__. 

```yaml
route:
  group_by: ['alertname']
  group_interval: 30s
  repeat_interval: 30s
  group_wait: 30s
  receiver: 'prometheus-msteams'

receivers:
- name: 'prometheus-msteams'
  webhook_configs: # https://prometheus.io/docs/alerting/configuration/#webhook_config 
  - send_resolved: true
    url: 'http://localhost:2000/alertmanager' # the prometheus-msteams proxy
```

> If you don't have Prometheus running yet and you wan't to try how this works,  
> try [stefanprodan's](https://github.com/stefanprodan) [Prometheus in Docker](https://github.com/stefanprodan/dockprom) to help you install a local Prometheus setup quickly in a single machine.

### Simulating a Prometheus Alerts to Teams Channel

Create the following json data as `prom-alert.json`.

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
curl -X POST -d @prom-alert.json http://localhost:2000/alertmanager
```

The teams channel should received a message.

## Sending Alerts to Multiple Teams Channel

### Creating the Configuration File

Create a __prometheus-msteams__ config file.

```yaml
connectors:
- high_priority_channel: "https://outlook.office.com/webhook/xxxx/aaa/bbb"
- low_priority_channel: "https://outlook.office.com/webhook/xxxx/aaa/ccc"
```

> __NOTE__: high_priority_channel and low_priority_channel are example handler names.  

When running as a docker container, mount the config file in the container and set the __CONFIG_FILE__ environment variable.

```bash
docker run -d -p 2000:2000 \
    --name="promteams" \
    -v /tmp/config.yml:/tmp/config.yml \
    -e CONFIG_FILE="/tmp/config.yml" \
    -e PROMTEAMS_DEBUG="true" \
    docker.io/bzon/prometheus-msteams:latest
```

When running as a binary, use the __--config__ flag.

```bash
PROMTEAMS_DEBUG="true" prometheus-msteams server \
	-l localhost \
	-p 2000 \
	--config /tmp/config.yml
```

This will create the request uri handlers __/high_priority_channel__ and __/low_priority_channel__.

### Setting up Prometheus Alert Manager

Considering the __prometheus-msteams config file__ settings, your Alert Manager would have a configuration like the following.

```yaml
route:
  group_by: ['alertname']
  group_interval: 30s
  repeat_interval: 30s
  group_wait: 30s
  receiver: 'low_priority_receiver'  # default/fallback request handler
  routes:
   - match:
     severity: critical
     receiver: high_priority_receiver
   - match:
     severity: warning
     receiver: low_priority_receiver

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

## Debugging

For debugging purposes, set the following environment variable `PROMTEAMS_DEBUG=true` to see the JSON body received from Prometheus and the JSON body created to be sent to Microsoft Teams.

```json
2018/03/19 11:21:23 Request received from Prometheus Alert Manager
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
2018/03/19 11:21:23 Created a card for Microsoft Teams
{
   "@type": "MessageCard",
   "@context": "http://schema.org/extensions",
   "themeColor": "8C1A1A",
   "summary": "Server High Memory usage",
   "title": "Prometheus Alert (firing)",
   "sections": [
     {
       "activityTitle": "[10.80.40.11 reported high memory usage with 23.28%.](http://docker.for.mac.host.internal:9093)",
       "facts": [
         {
           "name": "description",
           "value": "10.80.40.11 reported high memory usage with 23.28%."
         },
         {
           "name": "summary",
           "value": "Server High Memory usage"
         },
         {
           "name": "job",
           "value": "docker_nodes"
         },
         {
           "name": "monitor",
           "value": "master"
         },
         {
           "name": "severity",
           "value": "warning"
         },
         {
           "name": "alertname",
           "value": "high_memory_load"
         },
         {
           "name": "instance",
           "value": "10.80.40.11:9100"
         }
       ],
       "markdown": false
     }
   ]
 }
```

## Development and Testing

```bash
export GOTEST_TEAMS_INCOMING_WEBHOOK_URL="https://outlook.office.com/webhook/xxxx-xxxx-xxx"
make test
```

## Why?

Alertmanager doesn't support sending to Microsoft Teams out of the box. Fortunately, they allow you to use a generic [webhook_config](https://prometheus.io/docs/alerting/configuration/#webhook_config) for cases like this. This project was inspired from [idealista's](https://github.com/idealista/) [prom2teams](https://github.com/idealista/prom2teams) which was written in Python. 
