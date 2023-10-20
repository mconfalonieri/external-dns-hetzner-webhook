# ExternalDNS - Hetzner Webhook

**🛈 NOTE**: This Webhook was forked and modified from the [IONOS Webhook](https://github.com/ionos-cloud/external-dns-ionos-webhook)
to work with Hetzner. It contains parts from the original Hetzner provider that was removed from the main tree.


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

```shell
helm repo add bitnami https://charts.bitnami.com/bitnami
kubectl create secret generic hetzner-credentials --from-literal=api-key='<EXAMPLE_PLEASE_REPLACE>'

# create the helm values file
cat <<EOF > external-dns-hetzner-values.yaml
image:
  registry: ghcr.io
  repository: mconfalonieri/external-dns-webhook-provider
  tag: latest

provider: webhook

extraArgs:
  webhook-provider-url: http://localhost:8888

sidecars:
  - name: hetzner-webhook
    image: ghcr.io/mconfalonieri/external-dns-hetzner-webhook:$RELEASE_VERSION
    ports:
      - containerPort: 8888
        name: http
    livenessProbe:
      httpGet:
        path: /
        port: http
      initialDelaySeconds: 10
      timeoutSeconds: 5
    readinessProbe:
      httpGet:
        path: /
        port: http
      initialDelaySeconds: 10
      timeoutSeconds: 5
    env:
      - name: HETZNER_API_KEY
        valueFrom:
          secretKeyRef:
            name: hetzner-credentials
            key: api-key
      - name: SERVER_HOST
        value: "0.0.0.0" 
      - name: HETZNER_DEBUG
        value: "true"  
EOF
# install external-dns with helm
helm install external-dns-hetzner bitnami/external-dns -f external-dns-hetzner-values.yaml
```

The following environment variables are available:

| Variable        | Description                        | Notes                      |
| --------------- | ---------------------------------- | -------------------------- |
| HETZNER_API_KEY | Hetzner API token                  | Mandatory                  |
| DRY_RUN         | If set, changes won't be applied   | Default: `false`           |
| HETZNER_DEBUG   | Enables debugging messages         | Default: `false`           |
| BATCH_SIZE      | Number of zones per call           | Default: `100`, max: `100` |
| DEFAULT_TTL     | Default TTL if not specified       | Default: `7200`            |

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

## Development

The basic development tasks are provided by make. Run `make help` to see the available targets.

### Local deployment

The webhook can be deployed locally with a kind cluster. As a prerequisite, you need to install:

- [Docker](https://docs.docker.com/get-docker/),
- [Helm](https://https://helm.sh/ ) with the repos:

 ```shell
  helm repo add bitnami https://charts.bitnami.com/bitnami
  helm repo add mockserver https://www.mock-server.com
  helm repo update
  ```

- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

```shell
# setup the kind cluster and deploy external-dns with Hetzner webhook and a dns mockserver
./scripts/deploy_on_kind.sh

# check if the webhook is running
kubectl get pods -l app.kubernetes.io/name=external-dns -o wide

# trigger a DNS change e.g. with annotating the ingress controller service
kubectl -n ingress-nginx annotate service  ingress-nginx-controller "external-dns.alpha.kubernetes.io/internal-hostname=nginx.internal.example.org." 
 
# cleanup
./scripts/deploy_on_kind.sh clean
```
