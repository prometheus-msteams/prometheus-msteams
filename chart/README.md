## Installing the Chart

<!-- vim-markdown-toc GFM -->

* [Download the chart](#download-the-chart)
* [Prepare the Deployment configuration](#prepare-the-deployment-configuration)
* [Deploy to Kubernetes cluster](#deploy-to-kubernetes-cluster)
* [When using with Prometheus Operator](#when-using-with-prometheus-operator)
* [Helm Configuration](#helm-configuration)

<!-- vim-markdown-toc -->


### Download the chart

Clone this repository.

```bash
git clone https://github.com/bzon/prometheus-msteams
cd prometheus-msteams/chart
```

### Prepare the Deployment configuration

Create a helm values file to configure your Microsoft Teams channel connectors and customise the Kubernetes deployment.

```yaml
# config.yaml
---
replicaCount: 1
image:
  repository: bzon/prometheus-msteams 
  tag: v1.1.2
connectors:
- high_priority_channel: https://outlook.office.com/webhook/xxxx/xxxx 
- low_priority_channel: https://outlook.office.com/webhook/xxxx/xxxx
# extraEnvs is useful for adding extra environment variables such as proxy settings
extraEnvs:
  HTTP_PROXY: http://corporateproxy:8080
  HTTPS_PROXY: http://corporateproxy:8080
container:
  additionalArgs:
    - --max-idle-conns=100
    - --idle-conn-timeout=90s
    - --tls-handshake-timeout=30s
```

See [Helm Configuration](#helm-configuration) for reference.

Optionally you can customise the [Teams Message Card Template](./prometheus-msteams/card.tmpl). See [README](../README.md#customise-messages-to-ms-teams) for more information.

### Deploy to Kubernetes cluster

```bash
helm install --name prometheus-msteams ./prometheus-msteams --namespace monitoring -f config.yaml
```

### When using with Prometheus Operator

Please see [Prometheus Operator alerting docs](https://github.com/coreos/prometheus-operator/blob/master/Documentation/user-guides/alerting.md).

### Helm Configuration

| Parameter                | Description                                            | Default                                         |
| ---                      | ---                                                    | ---                                             |
| image.repository         | Image repository                                       | bzon/prometheus-msteams                         |
| image.tag                | Image tag                                              | v1.1.2                                          |
| image.pullPolicy         | Image pull policy                                      | Always                                          |
| extraEnvs                | Extra environment variables                            | {}                                              |
| connectors               | **Required.** Add your own Microsoft Teams connectors. | See [default](./prometheus-msteams/values.yaml) |
| service.port             | Service port                                           | 2000                                            |
| service.type             | Service type                                           | ClusterIP                                       |
| container.port           | Container port                                         | 2000                                            |
| container.additionalArgs | additional prometheus-msteams flags to use             | None                                            |
| resources                | CPU/memory resource requests/limits                    | See [default](./prometheus-msteams/values.yaml) |
| nodeSelector             | Labels for Node selector                               | {}                                              |
