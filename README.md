# ExternalDNS - Hetzner Webhook

[ExternalDNS](https://github.com/kubernetes-sigs/external-dns) is a Kubernetes
add-on for automatically DNS records for Kubernetes services using different
providers. By default, Kubernetes manages DNS records internally, but
ExternalDNS takes this functionality a step further by delegating the management
of DNS records to an external DNS provider such as this one. This webhook allows
you to manage your Hetzner domains inside your kubernetes cluster.

⚠️ If you are upgrading to 1.0.x from 0.6.x read the
[Upgrading from previous versions](#upgrading-from-previous-versions) section.

## Requirements

An
[API token](https://docs.hetzner.com/dns-console/dns/general/api-access-token/)
for the account managing your domains is required for this webhook to work
properly.

This webhook can be used in conjunction with **ExternalDNS v0.14.0 or higher**,
configured for using the webhook interface. Some examples for a working
configuration are shown in the next section.

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

## Upgrading from previous versions

### 0.x.x to 1.0.x

The configuration for previous versions are still compatible, but consider that
some warnings will be emitted if `HEALTH_HOST` and `HEALTH_PORT` are set. The
changes to be aware of are:

- `HEALTH_HOST` is deprecated in favor of `METRICS_HOST`;
- `HEALTH_PORT` is deprecated in favor of `METRICS_PORT`;
- the previous health/public socket is now called "metrics socket" in conformity
  to ExternalDNS terminology, and now supports some additional endpoints:
  
  - `/metrics` and
  - `/healthz`;

  their description can be found in the [Metrics socket](#metrics-socket)
  section.
  


## Environment variables

The following environment variables can be used for configuring the application.

### Hetzner DNS API calls configuration

These variables control the behavior of the webhook when interacting with
Hetzner DNS API.

| Variable        | Description              | Notes                      |
| --------------- | -------------------------| -------------------------- |
| HETZNER_API_KEY | Hetzner API token        | Mandatory                  |
| BATCH_SIZE      | Number of zones per call | Default: `100`, max: `100` |
| DEFAULT_TTL     | Default record TTL       | Default: `7200`            |

### Test and debug

These environment variables are useful for testing and debugging purposes.

| Variable        | Description                      | Notes            |
| --------------- | -------------------------------- | ---------------- |
| DRY_RUN         | If set, changes won't be applied | Default: `false` |
| HETZNER_DEBUG   | Enables debugging messages       | Default: `false` |

### Socket configuration

These variables control the sockets that this application listens to.

| Variable        | Description                      | Notes                |
| --------------- | -------------------------------- | -------------------- |
| WEBHOOK_HOST    | Webhook hostname or IP address   | Default: `localhost` |
| WEBHOOK_PORT    | Webhook port                     | Default: `8888`      |
| METRICS_HOST    | Metrics hostname                 | Default: `0.0.0.0`   |
| METRICS_PORT    | Metrics port                     | Default: `8080`      |
| READ_TIMEOUT    | Sockets' read timeout in ms      | Default: `60000`     |
| WRITE_TIMEOUT   | Sockets' write timeout in ms     | Default: `60000`     |

Please notice that the following variables were **deprecated**:

| Variable    | Description                      |
| ----------- | -------------------------------- |
| HEALTH_HOST | Metrics hostname (deprecated)    |
| HEALTH_PORT | Metrics port (deprecated)        |


### Domain filtering

Additional environment variables for domain filtering. When used, this webhook
will be able to work only on domains matching the filter.

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
| Metrics     | `METRICS_HOST`:`METRICS_PORT` |

The environment variables controlling the socket addresses are not meant to be
changed, under normal circumstances, for the reasons explained in
[Tweaking the configuration](tweaking-the-configuration).
The endpoints
[expected by ExternalDNS](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/tutorials/webhook-provider.md)
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

| Endpoint           | * | Purpose                                            |
| ------------------ | - | -------------------------------------------------- |
| `/health`          |   | Implements the liveness probe                      |
| `/ready`           |   | Implements the readiness probe                     |
| `/healthz`         | * | Implements a combined liveness and readiness probe |
| `/metrics`         | * | Exposes the available metrics                      |

Please check the [Exposed metrics](#exposed-metrics) section for more
information.

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

## Exposed metrics

The following metrics related to the API calls towards Hetzner are available
for scraping.

| Name                         | Type      | Labels   | Description                                              |
| ---------------------------- | --------- | -------- | -------------------------------------------------------- |
| `successful_api_calls_total` | Counter   | `action` | The number of successful Hetzner API calls               |
| `failed_api_calls_total`     | Counter   | `action` | The number of Hetzner API calls that returned an error   |
| `filtered_out_zones`         | Gauge     | _none_   | The number of zones excluded by the domain filter        |
| `api_delay_count`            | Histogram | `action` | Histogram of the delay (ms) when calling the Hetzner API |

The label `action` can assume one of the following values, depending on the
Hetzner API endpoint called:

- `get_zones`
- `get_records`
- `create_record`
- `delete_record`
- `update_record`

Please notice that in some cases an _update_ request from ExternalDNS will be
transformed into a `delete_record` and subsequent `create_record` calls by this
webhook.


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