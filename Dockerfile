FROM alpine:3.13

RUN apk --no-cache add ca-certificates tini tzdata

LABEL description="A lightweight Go Web Server that accepts POST alert message from Prometheus Alertmanager and sends it to Microsoft Teams Channels using an incoming webhook url."

COPY ./default-message-card.tmpl /default-message-card.tmpl
COPY bin/prometheus-msteams-linux-amd64 /promteams

ENTRYPOINT ["/sbin/tini", "--", "/promteams"]

EXPOSE 2000
