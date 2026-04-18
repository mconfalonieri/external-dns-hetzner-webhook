# Endpoints

This process exposes several endpoints, that will be available through these
sockets:

| Socket name | Socket address                |
| ----------- | ----------------------------- |
| Webhook     | `WEBHOOK_HOST`:`WEBHOOK_PORT` |
| Metrics     | `METRICS_HOST`:`METRICS_PORT` |

The environment variables controlling the socket addresses are not meant to be
changed, under normal circumstances, for the reasons explained in
[Tweaking the configuration](tweaking-the-configuration).
The endpoints
[expected by ExternalDNS](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/tutorials/webhook-provider.md)
are marked with *.

## Webhook socket

All these endpoints are
[required by ExternalDNS](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/tutorials/webhook-provider.md).

| Endpoint           | Purpose                                        |
| ------------------ | ---------------------------------------------- |
| `/`                | Initialization and `DomainFilter` negotiations |
| `/record`          | Get and apply records                          |
| `/adjustendpoints` | Adjust endpoints before submission             |

## Metrics socket

ExternalDNS doesn't have functional requirements for this endpoint, but some
of them are
[recommended](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/tutorials/webhook-provider.md).
In this table those endpoints are marked with  __*__.

| Endpoint           | * | Purpose                                            |
| ------------------ | - | -------------------------------------------------- |
| `/health`          |   | Implements the liveness probe                      |
| `/ready`           |   | Implements the readiness probe                     |
| `/healthz`         | * | Implements a combined liveness and readiness probe |
| `/metrics`         | * | Exposes the available metrics                      |

Please check the [Exposed metrics](#exposed-metrics) section for more
information.
