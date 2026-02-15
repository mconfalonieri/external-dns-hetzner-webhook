# ExternalDNS - UNOFFICIAL Hetzner Webhook

> [!IMPORTANT]
> Support for the legacy DNS is going to be discontinued by Hetzner in May 2026.
> The legacy provider will be pulled from this provider in version 1.0.0.
> No new features will be added to the legacy DNS driver and only important
> bugfixes will be backported.

> [!NOTE]
> The latest version is **v0.11.0**.

[ExternalDNS](https://github.com/kubernetes-sigs/external-dns) is a Kubernetes
add-on for automatically DNS records for Kubernetes services using different
providers. By default, Kubernetes manages DNS records internally, but
ExternalDNS takes this functionality a step further by delegating the management
of DNS records to an external DNS provider such as this one. This webhook allows
you to manage your Hetzner domains inside your kubernetes cluster.

This webhook supports both the old DNS API and the new Cloud DNS interface.

> [!TIP]
> If you are upgrading to **0.11.x** from previous versions read the
> [Upgrading from previous versions](#upgrading-from-previous-versions) section.


## Requirements

This webhook can be used in conjunction with **ExternalDNS v0.19.0 or higher**,
configured for using the webhook interface. Some examples for a working
configuration are shown in the next section.

### Legacy DNS API
A [DNS API token](https://docs.hetzner.com/dns-console/dns/general/api-access-token/)
for the account managing your domains is required for this webhook to work
properly when the **USE_CLOUD_API** environment variable is set to `false` or
not set.

### Cloud API

A [Cloud API token](https://docs.hetzner.com/cloud/api/getting-started/generating-api-token/)
is required when the new Cloud API is in use (the **USE_CLOUD_API** environment
variable is set to `true`).

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
      tag: v0.11.0
    env:
      - name: HETZNER_API_KEY
        valueFrom:
          secretKeyRef:
            name: hetzner-credentials
            key: api-key
    livenessProbe:
      httpGet:
        path: /health
        port: http-webhook
      initialDelaySeconds: 10
      timeoutSeconds: 5
    readinessProbe:
      httpGet:
        path: /ready
        port: http-webhook
      initialDelaySeconds: 10
      timeoutSeconds: 5

extraArgs:
  - "--txt-prefix=reg-%{record_type}-"
```

And then:

```shell
# install external-dns with helm
helm install external-dns-hetzner external-dns/external-dns -f external-dns-hetzner-values.yaml -n external-dns
```

### Using the Bitnami chart

> [!NOTE]
> The Bitnami distribution model changed and most features are now paid for.

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
provider: webhook
policy: sync
extraArgs:
  webhook-provider-url: http://localhost:8888
  txt-prefix: "reg-%{record_type}-"

sidecars:
  - name: hetzner-webhook
    image: ghcr.io/mconfalonieri/external-dns-hetzner-webhook:v0.11.0
    ports:
      - containerPort: 8888
        name: webhook
      - containerPort: 8080
        name: http-wh-metrics
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

## Hetzner labels

Hetzner labels are supported from version **0.8.0** as provider-specific
annotations. This feature has some additional requirements to work properly:

- External DNS in use must be **0.19.0** or higher
- the zone must be migrated to Hetzner Cloud console
- **USE_CLOUD_API** must be set to `true`
- **HETZNER_API_TOKEN** must refer to a Cloud API token

The labels are set with an annotation prefixed with:
`external-dns.alpha.kubernetes.io/webhook-hetzner-label-`.

For example, if we want to set these labels:

| Label      | Value      |
| ---------- | ---------- |
| it.env     | production |
| department | education  |

The annotation syntax will be:

```yaml
  external-dns.alpha.kubernetes.io/webhook-hetzner-label-it.env: production
  external-dns.alpha.kubernetes.io/webhook-hetzner-label-department: education
```

This kind of label:

| Label        | Value |
| ------------ | ----- |
| prefix/label | value |

requires an escape sequence for the slash part. By default this will be:
`--slash--`, so the label will be written as:

```yaml
  external-dns.alpha.kubernetes.io/webhook-hetzner-label-prefix--slash--label: value
```

This can be changed using the **SLASH_ESC_SEQ** environment variable.

## Bulk mode

The Cloud API now supports a new way of updating the records for a zone called
**bulk mode**. This mode is activated by setting the `BULK_MODE` environment
variable to `true`. It works by exporting the zonefile, editing it and then
uploading the modified version. It is meant to be used in environments with a
high number of record changes per zone and a relatively long interval between
the updates, a combination that could cause the exhaustion of the permitted API
calls.

> [!WARNING]
> Beware that this method of updating the records is potentially destructive
> and subject to "race conditions" if manual edits are applied while the zone
> is being updated. Theoretically, unsupported records won't be affected, but
> this method is to be considered **HIGHLY EXPERIMENTAL**, and bugs are likely
> to be found.

It comes with some limitations.

  1. [Hetzner labels](#hetzner-labels) are not supported, as there is no way to
     import them in the zonefile. Record comments are unsupported as well.
  2. All the records must be **not protected** as they will all be overwritten
     during the import operation, **including the SOA**. This is why the bulk
     mode should be used with care.
  3. The SOA serial number is updated on each import, but the logic of the
     serial number only accepts the standard 10-digits serial number and will
     refuse to update it if the serial of the day is 99. Most configurations
     will be OK with this limitation.
  4. If the zones managed by this webhook are also manipuilated by other
     software the following situation, although unlikely, could happen:
       
       1. the webhook downloads the zonefile for a zone
       2. the other software manipulates some record on the same zone
       3. the webhook uploads the zonefile, missing the changes applied by the
          other software
       4. since the upload rewrites the zonefile, those changes are now lost.
  
Please check the [Zone file import](https://docs.hetzner.cloud/reference/cloud#tag/zones/zone-file-import)
section of the Hetzner documentation for more details.

## Upgrading from previous versions

### 0.10.x to 0.11.x

The configuration is compatible with previous versions; however the
**DEFAULT_TTL** parameter was removed and this environment variable will
therefore not affect the configuration. The default TTL will be the one defined
in the zone.

A **BULK_MODE** parameter was added. When set to `true`, the webhook will
export and manipulate the zonefiles instead of the single recordsets. This will
reduce the API calls when updating zones with lots of changes and a relatively
long interval.

> [WARNING]
> The bulk mode is experimental and comes with some limitations. Please read
> the [Bulk mode](#bulk-mode) section before activating it.
>


### 0.9.x to 0.10.x

The configuration is fully compatible. There is a new configuration parameter
**ZONE_CACHE_TTL** that controls the TTL of the newly implemented zones
cache, that is aimed to reduce API calls. The parameter is expressed in
seconds.

### 0.8.x to 0.9.x

The configuration is fully compatible. A new configuration parameter
**MAX_FAIL_COUNT** was added to control the webhook behavior in case of
repeated failed attempts to retrieve the records. If this parameter is set to
a value strictly greater than zero, the webhook will shut down after the
configured number of attempts. The default is `-1` (shutdown disabled).

MX records are now supported for both the Cloud API (since **v0.9.1**) and the
Legacy DNS API (since **v0.9.2**).

### 0.7.x to 0.8.x

The configuration is still compatible, however some changes were introduced that
might be worth to check out:

- the new Cloud API is now supported through the **USE_CLOUD_API** environment
  variable and using a Cloud API token for **HETZNER_API_TOKEN**;
- [Hetzner labels](#hetzner-labels) are available when the new Cloud API is in use;
- The minimum ExternalDNS version is now **0.19.0** as the label system and the
  Cloud API are untested with previous versions.

Please notice that the Cloud API requires migrating the DNS zones to the new
DNS tab in Hetzner console.

### 0.6.x to 0.7.x

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

| Variable        | Description                            | Notes                      |
| --------------- | -------------------------------------- | -------------------------- |
| HETZNER_API_KEY | Hetzner API token                      | Mandatory                  |
| BATCH_SIZE      | Number of zones per call               | Default: `100`, max: `100` |
| USE_CLOUD_API   | Use the new cloud API                  | Default: `false`           |
| SLASH_ESC_SEQ   | Escape sequence for label annotations  | Default: `--slash--`       |
| MAX_FAIL_COUNT  | Number of failed calls before shutdown | Default: `-1` (disabled)   |
| ZONE_CACHE_TTL  | TTL for the zone cache in seconds      | Default: `0` (disabled)    |
| BULK_MODE       | Enables bulk mode                      | Default: `false`           |

> [!IMPORTANT]
> Please notice that when **USE_CLOUD_API** is set to `true`, the token stored 
> in **HETZNER_API_KEY** must be a Hetzner Cloud token, NOT the classic DNS one.

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

| Variable    | Description                            |
| ----------- | -------------------------------------- |
| HEALTH_HOST | Metrics hostname (deprecated)          |
| HEALTH_PORT | Metrics port (deprecated)              |
| DEFAULT_TTL | The default TTL is taken from the zone |


### Domain filtering

Additional environment variables for domain filtering. When used, this webhook
will be able to work only on domains (zones) matching the filter.

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

- if `WEBHOOK_HOST` and `METRICS_HOST` are set to the same address/hostname or
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
| `skipped_records`            | Gauge     | `zone`   | The number of skipped records per domain                 |
| `api_delay_hist`             | Histogram | `action` | Histogram of the delay (ms) when calling the Hetzner API |

The label `action` can assume one of the following values, depending on the
Hetzner API endpoint called.

The actions supported by the legacy DNS provider are:

- `get_zones`
- `get_records`
- `create_record`
- `delete_record`
- `update_record`

The actions supported by the Cloud API provider are:

- `get_zones`
- `get_rrsets`
- `create_rrset`
- `update_rrset_ttl`
- `update_rrset_records`
- `update_rrset` (this is the method used to update labels)
- `delete_rrset`

The label `zone` can assume one of the zone names as its value.

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
| [sgaluza](https://github.com/sgaluza)        | support for MX records        |
