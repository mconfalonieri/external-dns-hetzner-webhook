# ExternalDNS - Hetzner Webhook

⚠️  This software is experimental and **NOT FIT FOR PRODUCTION USE!**

**🛈 NOTE**: This Webhook was forked and modified from the [IONOS Webhook](https://github.com/ionos-cloud/external-dns-ionos-webhook)
to work with Hetzner. It also contains huge parts from the original Hetzner provider that was removed from the main tree.

ExternalDNS is a Kubernetes add-on for automatically managing
Domain Name System (DNS) records for Kubernetes services by using different DNS providers.
By default, Kubernetes manages DNS records internally,
but ExternalDNS takes this functionality a step further by delegating the management of DNS records to an external DNS
provider such as Hetzner.
Therefore, the Hetzner webhook allows to manage your
Hetzner domains inside your kubernetes cluster with [ExternalDNS](https://github.com/kubernetes-sigs/external-dns).

To use ExternalDNS with Hetzner, you need your Hetzner API token of the account managing
your domains.
For detailed technical instructions on how the Hetzner webhook is deployed using the Bitnami Helm charts for ExternalDNS,
see[deployment instructions](#kubernetes-deployment).

## Kubernetes Deployment

The Hetzner webhook is provided as a regular Open Container Initiative (OCI) image released in
the [GitHub container registry](https://github.com/mconfalonieri/external-dns-hetzner-webhook/pkgs/container/external-dns-hetzner-webhook).
The deployment can be performed in every way Kubernetes supports.
The following example shows the deployment as
a [sidecar container](https://kubernetes.io/docs/concepts/workloads/pods/#workload-resources-for-managing-pods) in the
ExternalDNS pod
using the [Bitnami Helm charts for ExternalDNS](https://github.com/bitnami/charts/tree/main/bitnami/external-dns).

⚠️  Webhooks are still an experimental feature of External DNS. The image used in this configuration example
was created by me from the External DNS' master branch to test the webhook.
This notice and configuration file will be updated once a webhook version of External DNS will be released.

```shell
helm repo add bitnami https://charts.bitnami.com/bitnami
kubectl create secret generic hetzner-credentials --from-literal=api-key='<EXAMPLE_PLEASE_REPLACE>'
```
You can then create the helm values file, for example
`external-dns-hetzner-values.yaml`:

```yaml
image:
  registry: docker.io
  repository: mconfalonieri/external-dns
  tag: v0.13.6-142-gd8f408b8

provider: webhook

extraArgs:
  webhook-provider-url: http://localhost:8888
  txt-prefix: reg-

sidecars:
  - name: hetzner-webhook
    image: ghcr.io/mconfalonieri/external-dns-hetzner-webhook:v0.5.0
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
helm install external-dns-hetzner bitnami/external-dns -f external-dns-hetzner-values.yaml
```

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
| HEALTH_HOST     | Liveness and readiness hostname  | Default: `0.0.0.0`         |
| HEALTH_PORT     | Liveness and readiness port      | Default: `8080`            |
| READ_TIMEOUT    | Servers' read timeout in ms      | Default: `60000`           |
| WRITE_TIMEOUT   | Servers' write timeout in ms     | Default: `60000`           |

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

## Tweaking the configuration

While tweaking the configuration, there are some points to take into
consideration:

- if you are using `--registry=txt` (default) for `external-dns`, remember to
  set `--txt-prefix=<prefix>` too, or you will receive a 422 response back when
  setting one of the required `TXT` records;
- if `WEBHOOK_HOST` and `HEALTH_HOST` are set to the same address/hostname or
  one of them is set to `0.0.0.0` remember to use different ports.

## Development

The basic development tasks are provided by make. Run `make help` to see the
available targets.
