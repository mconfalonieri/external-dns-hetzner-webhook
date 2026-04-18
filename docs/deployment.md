# Kubernetes deployment

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

## Using the ExternalDNS chart

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
      tag: {{ .Version }}
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
  - "--txt-prefix=reg-%{record_type}."
```

And then:

```shell
# install external-dns with helm
helm install external-dns-hetzner external-dns/external-dns -f external-dns-hetzner-values.yaml -n external-dns
```

## Using the Bitnami chart

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
  txt-prefix: "reg-%{record_type}."

sidecars:
  - name: hetzner-webhook
    image: ghcr.io/mconfalonieri/external-dns-hetzner-webhook:{{ .Version }}
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
