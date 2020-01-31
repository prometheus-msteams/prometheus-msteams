## Development

### Workflow

```
make dep
make lint
make test
```

### Tracing

Run a Jaeger instance locally.


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



