# Contributing

prometheus-msteams is written in Go so ensure to install the latest stable version available.

You can also [join](https://teams.microsoft.com/l/team/19%3a8f7647537f14400cbd7032d058a05648%40thread.skype/conversations?groupId=c49097a8-ba31-4e64-b482-a5626e7cac18&tenantId=b12d4011-2ea0-4377-a99b-35c565546afd) the Microsoft Teams Channel for discussions.

### Dependency

We use go mod for dependency management.

```
make dep
```

### Code Quality

Install [golangci-lint](https://github.com/golangci/golangci-lint).

```
make lint
```

### Testing

Just run `make test`.

Run integration test to a Microsoft Teams channel.

```
export INTEGRATION_TEST_WEBHOOK_URL=<webhook url>
go test -v ./e2e/...
```

### Tracing

Debugging is much easier with tracing.

First, run a local instance of Jaeger.

```yaml
version: '2'

services:
  jaeger:
    container_name: jaeger
    image: jaegertracing/all-in-one:latest
    environment:
      - COLLECTOR_ZIPKIN_HTTP_PORT=9411
    ports:
      - "5575:5575/udp"
      - "6831:6831/udp"
      - "6832:6832/udp"
      - "5778:5778"
      - "16686:16686"
      - "14268:14268"
      - "9411:9411"
```

Then, use the `-jaeger-trace=true` and `-jaeger-agent=localhost:6831` to start the server.

Finally, access http://localhost:16686 and you should see the traces in Jaeger.

![](https://user-images.githubusercontent.com/19391568/73496892-d896e580-43b9-11ea-8d5f-150ed533665e.png)



