FROM alpine:latest as certs

RUN apk --no-cache add ca-certificates tzdata

FROM scratch
LABEL description="A lightweight Go Web Server that accepts POST alert message from Prometheus Alertmanager and sends it to Microsoft Teams Channels using an incoming webhook url."
EXPOSE 2000

# Copy required cert and zoneinfo from previous stage
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ 
COPY --from=certs /usr/share/zoneinfo /usr/share/zoneinfo

COPY ./default-message-card.tmpl /default-message-card.tmpl
COPY ./default-message-workflow-card.tmpl /default-message-workflow-card.tmpl
COPY bin/prometheus-msteams-linux-amd64 /promteams

ENTRYPOINT ["/promteams"]
