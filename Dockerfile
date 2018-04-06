FROM alpine

RUN apk --no-cache add ca-certificates && update-ca-certificates

ARG VERSION

ENV VERSION=$VERSION \
    MARKDOWN_ENABLED=yes \
    CONFIG_FILE="$HOME/.prometheus-msteams.yaml" \
    TEAMS_REQUEST_URI="/alertmanager" \
    TEAMS_INCOMING_WEBHOOK_URL="" \
    PROMTEAMS_DEBUG="true"

LABEL summary=$SUMMARY \
      description="A lightweight Go Web Server that accepts POST alert message from Prometheus Alertmanager and sends it to Microsoft Teams Channels using an incoming webhook url." \
      version=$VERSION

ADD prometheus-msteams-linux-amd64 /bin/promteams
COPY docker/cmd.sh /bin/container-cmd

CMD /bin/container-cmd

EXPOSE 2000
