# ExternalDNS - Hetzner Webhook

![Tests](https://camo.githubusercontent.com/1005474327f297a66493a98db94bf2b4a3fb9fce5a9f99a9c324cf13fa48d247/68747470733a2f2f696d672e736869656c64732e696f2f62616467652f74657374732d3135362532307061737365642d73756363657373)

[ExternalDNS](https://github.com/kubernetes-sigs/external-dns) is a Kubernetes
add-on for automatically managing Domain Name System (DNS) records for
Kubernetes services using different DNS providers. By default, Kubernetes
manages DNS records internally, but ExternalDNS takes this functionality a step
further by delegating the management of DNS records to an external DNS provider
such as this one. This webhook allows you to manage your Hetzner domains inside
your kubernetes cluster.

## Requirements

An
[API token](https://docs.hetzner.com/dns-console/dns/general/api-access-token/)
for the account managing your domains is required for this webhook to work
properly.

⚠️ This webhook requires at least ExternalDNS v0.14.0.

## Kubernetes Deployment

The Hetzner webhook is provided as a regular Open Container Initiative (OCI)
image released in the
[GitHub container registry](https://github.com/mconfalonieri/external-dns-hetzner-webhook/pkgs/container/external-dns-hetzner-webhook).
The deployment can be performed in every way Kubernetes supports.

Here are provided examples using the
[External DNS chart](#using-the-externaldns-chart) and the
[Bitnami chart](#using-the-bitnami-chart).

In either case, a secret that stores the Hetzner API key is required:

```yaml
kubectl create secret generic hetzner-credentials --from-literal=api-key='<EXAMPLE_PLEASE_REPLACE>' -n external-dns
```

### Using the ExternalDNS chart

Skip this step if you already have the ExternalDNS repository added:

```shell
helm repo add external-dns https://kubernetes-sigs.github.io/external-dns/
```

Update your helm chart repositories:

```shell
helm repo update
```

You can then create the helm values file, for example
`external-dns-hetzner-values.yaml`:

```yaml
namespace: external-dns
policy: sync
provider:
  name: webhook
  webhook:
    image:
      repository: ghcr.io/mconfalonieri/external-dns-hetzner-webhook
      tag: v0.6.0
    env:
      - name: HETZNER_API_KEY
        valueFrom:
          secretKeyRef:
            name: hetzner-credentials
            key: api-key
    livenessProbe:
      httpGet:
        path: /health
        port: http-wh-metrics
      initialDelaySeconds: 10
      timeoutSeconds: 5
    readinessProbe:
      httpGet:
        path: /ready
        port: http-wh-metrics
      initialDelaySeconds: 10
      timeoutSeconds: 5

extraArgs:
  - "--txt-prefix=reg-%{record_type}-"
```

And then:

```shell
# install external-dns with helm
helm install external-dns-hetzner external-dns/external-dns -f external-dns-hetzner-values.yaml --version 1.14.3 -n external-dns
```

### Using the Bitnami chart

Skip this step if you already have the Bitnami repository added:

```shell
helm repo add bitnami https://charts.bitnami.com/bitnami
```

Update your helm chart repositories:

```shell
helm repo update
```

You can then create the helm values file, for example
`external-dns-hetzner-values.yaml`:

```yaml
image:
  registry: registry.k8s.io
  repository: external-dns/external-dns
  tag: v0.14.0

provider: webhook

extraArgs:
  webhook-provider-url: http://localhost:8888
  txt-prefix: "reg-%{record_type}-"

sidecars:
  - name: hetzner-webhook
    image: ghcr.io/mconfalonieri/external-dns-hetzner-webhook:v0.6.0
    ports:
      - containerPort: 8888
        name: webhook
      - containerPort: 8080
        name: http
    livenessProbe:
      httpGet:
        path: /health
        port: http
      initialDelaySeconds: 10
      timeoutSeconds: 5
    readinessProbe:
      httpGet:
        path: /ready
        port: http
      initialDelaySeconds: 10
      timeoutSeconds: 5
    env:
      - name: HETZNER_API_KEY
        valueFrom:
          secretKeyRef:
            name: hetzner-credentials
            key: api-key
```

And then:

```shell
# install external-dns with helm
helm install external-dns-hetzner bitnami/external-dns -f external-dns-hetzner-values.yaml -n external-dns
```

## Environment variables

The following environment variables are available:

| Variable        | Description                      | Notes                      |
| --------------- | -------------------------------- | -------------------------- |
| HETZNER_API_KEY | Hetzner API token                | Mandatory                  |
| DRY_RUN         | If set, changes won't be applied | Default: `false`           |
| HETZNER_DEBUG   | Enables debugging messages       | Default: `false`           |
| BATCH_SIZE      | Number of zones per call         | Default: `100`, max: `100` |
| DEFAULT_TTL     | Default TTL if not specified     | Default: `7200`            |
| WEBHOOK_HOST    | Webhook hostname or IP address   | Default: `localhost`       |
| WEBHOOK_PORT    | Webhook port                     | Default: `8888`            |
| HEALTH_HOST     | Metrics hostname                 | Default: `0.0.0.0`         |
| HEALTH_PORT     | Metrics port                     | Default: `8080`            |
| READ_TIMEOUT    | Sockets' read timeout in ms      | Default: `60000`           |
| WRITE_TIMEOUT   | Sockets' write timeout in ms     | Default: `60000`           |

Additional environment variables for domain filtering:

| Environment variable           | Description                        |
| ------------------------------ | ---------------------------------- |
| DOMAIN_FILTER                  | Filtered domains                   |
| EXCLUDE_DOMAIN_FILTER          | Excluded domains                   |
| REGEXP_DOMAIN_FILTER           | Regex for filtered domains         |
| REGEXP_DOMAIN_FILTER_EXCLUSION | Regex for excluded domains         |

If the `REGEXP_DOMAIN_FILTER` is set, the following variables will be used to
build the filter:

 - REGEXP_DOMAIN_FILTER
 - REGEXP_DOMAIN_FILTER_EXCLUSION

 otherwise, the filter will be built using:

 - DOMAIN_FILTER
 - EXCLUDE_DOMAIN_FILTER

## Endpoints

This process exposes several endpoints, that will be available through these
sockets:

| Socket name | Socket address                |
| ----------- | ----------------------------- |
| Webhook     | `WEBHOOK_HOST`:`WEBHOOK_PORT` |
| Metrics     | `HEALTH_HOST`:`HEALTH_PORT`   |

**Note**: the "Health" socket was renamed to "Metrics" to conform to
ExternalDNS terminology, but the prefix is still `HEALTH` for compatibility
with the previous versions of this webhook.

The environment variables controlling the socket addresses are not meant to be
changed, under normal circumstances, for the reasons explained in
[Tweaking the configuration](tweaking-the-configuration).
The endpoints [expected by ExternalDNS](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/tutorials/webhook-provider.md)
are marked with *.

### Webhook socket

All these endpoints are
[required by ExternalDNS](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/tutorials/webhook-provider.md).

| Endpoint           | Purpose                                        |
| ------------------ | ---------------------------------------------- |
| `/`                | Initialization and `DomainFilter` negotiations |
| `/record`          | Get and apply records                          |
| `/adjustendpoints` | Adjust endpoints before submission             |

### Metrics socket

ExternalDNS doesn't have functional requirements for this endpoint, but some
of them are
[recommended](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/tutorials/webhook-provider.md).
In this table those endpoints are marked with  __*__.

| Endpoint           | * | Purpose                                                               |
| ------------------ | - | --------------------------------------------------------------------- |
| `/health`          |   | Implements the liveness probe                                         |
| `/ready`           |   | Implements the readiness probe                                        |
| `/healthz`         | * | Implements the liveness and readiness probe                           |
| `/metrics`         | * | Exposes the [Open Metrics](https://github.com/prometheus/OpenMetrics) |

## Tweaking the configuration

While tweaking the configuration, there are some points to take into
consideration:

- if `WEBHOOK_HOST` and `HEALTH_HOST` are set to the same address/hostname or
  one of them is set to `0.0.0.0` remember to use different ports. Please note
  that it **highly recommendend** for `WEBHOOK_HOST` to be `localhost`, as
  any address reachable from outside the pod might be a **security issue**;
  besides this, changing these would likely need more tweaks than just setting
  the environment variables. The default settings are compatible with the
  [ExternalDNS assumptions](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/tutorials/webhook-provider.md);
- if your records don't get deleted when applications are uninstalled, you
  might want to verify the policy in use for ExternalDNS: if it's `upsert-only`
  no deletion will occur. It must be set to `sync` for deletions to be
  processed. Please check that `external-dns-hetzner-values.yaml` include:

  ```yaml
  policy: sync
  ```
- the `--txt-prefix` parameter should really include: `%{record_type}`, as any
  other value will cause a weird duplication of database records. Change the
  value provided in the sample configuration only if you really know what are
  you doing.

## Development

The basic development tasks are provided by make. Run `make help` to see the
available targets.

## Credits

This Webhook was forked and modified from the [IONOS Webhook](https://github.com/ionos-cloud/external-dns-ionos-webhook)
to work with Hetzner. It also contains huge parts from [DrBu7cher's Hetzner provider](https://github.com/DrBu7cher/external-dns/tree/readding_hcloud_provider).

### Contributors

| Name                                         | Contribution                  |
| -------------------------------------------- | ----------------------------- |
| [DerQue](https://github.com/DerQue)          | local CNAME fix               |
| [sschaeffner](https://github.com/sschaeffner)| build configuration for arm64 |