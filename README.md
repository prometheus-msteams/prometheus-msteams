# Overview

A lightweight Go Web Server that accepts POST alert message from Prometheus Alertmanager and sends it to Microsoft Teams Channels using an incoming webhook url.

## Docker Installation

```bash
docker run -d -p 2000:2000 \
    --name="promteams" \
    -e MARKDOWN_ENABLED=yes \
    -e TEAMS_INCOMING_WEBHOOK_URL="https://outlook.office.com/webhook/xxxx-xxxx-xxx" \
    docker.io/bzon/prometheus-msteams:latest
```

## Go Installation

```bash
go build
./prometheus-msteams server --help
./prometheus-msteams server -l localhost -p 2000 -w "https://outlook.office.com/webhook/xxxx-xxxx-xxx"
```

## Alert Manager Configuration

You can try [stefanprodan's](https://github.com/stefanprodan) [Prometheus in Docker](https://github.com/stefanprodan/dockprom) to help you setup your Sandbox environment quickly.

Your Alertmanager would have a configuration like this.

```yaml
route:
    group_by: ['alertname']
    group_interval: 30s
    repeat_interval: 30s
    group_wait: 30s
    receiver: 'prometheus-msteams'

receivers:
    - name: 'prometheus-msteams'
      webhook_configs:
          - send_resolved: true
            url: 'http://localhost:2000/alertmanager'
```

## Debugging

For debugging purposes, I'm encoding and writing Request Body to `os.Stdout` so we can see the POST from Alertmanager and the POST to MS Teams.

```json
2018/03/05 03:42:16 Request received
{"version":"4","groupKey":"{}:{alertname=\"jenkins_down\"}","status":"firing","receiver":"teams_proxy","groupLabels":{"alertname":"jenkins_down"},"commonLabels":{"alertname":"jenkins_down","monitor":"docker-host-alpha","name":"jenkins","severity":"critical"},"commonAnnotations":{"description":"Jenkins container is down for more than 30 seconds.","summary":"Jenkins down"},"externalURL":"http://67067274c8d9:9093","alerts":[{"labels":{"alertname":"jenkins_down","monitor":"docker-host-alpha","name":"jenkins","severity":"critical"},"annotations":{"description":"Jenkins container is down for more than 30 seconds.","summary":"Jenkins down"},"startsAt":"2018-03-04T16:04:36.125572966Z","endsAt":"0001-01-01T00:00:00Z"}]}
2018/03/05 03:42:16 Creating a card
{"@type":"MessageCard","@context":"http://schema.org/extensions","themeColor":"8C1A1A","summary":"Jenkins down","title":"Prometheus Alert (firing)","sections":[{"activityTitle":"[Jenkins container is down for more than 30 seconds.](http://67067274c8d9:9093)","facts":[{"name":"alertname","value":"jenkins_down"},{"name":"monitor","value":"docker-host-alpha"},{"name":"name","value":"jenkins"},{"name":"severity","value":"critical"}],"markdown":true}]}
2018/03/05 03:42:16 Sending the card
2018/03/05 03:42:17 Total Card sent since uptime: 2
```

## Why

Alertmanager doesn't support sending to Microsoft Teams out of the box. Fortunately, they allow you to use a generic [webhook_config](https://prometheus.io/docs/alerting/configuration/#webhook_config) for cases like this. This project was inspired from [idealista's](https://github.com/idealista/) [prom2teams](https://github.com/idealista/prom2teams) which was written in Python. 
