FROM alpine

RUN apk --no-cache add ca-certificates && update-ca-certificates

ARG VERSION

ENV VERSION=$VERSION

LABEL summary=$SUMMARY \
      description="A lightweight Go Web Server that accepts POST alert message from Prometheus Alertmanager and sends it to Microsoft Teams Channels using an incoming webhook url." \
      version=$VERSION

ADD prometheus-msteams-linux-amd64 /bin/promteams

CMD /bin/promteams server -p 2000 -w "${TEAMS_INCOMING_WEBHOOK_URL}"

EXPOSE 2000
