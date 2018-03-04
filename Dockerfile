FROM alpine

RUN apk --no-cache add ca-certificates && update-ca-certificates

LABEL description="Prometheus alert receiver." \
      version=${VERSION} \
      branch=${BRANCH} \
      commit=${COMMIT}

ADD promteams-linux-amd64 /bin/promteams

CMD /bin/promteams server -p 2000 -w "${TEAMS_INCOMING_WEBHOOK_URL}"

EXPOSE 8080
