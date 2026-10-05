# Kubernetes deployment

The Hetzner webhook is provided as source repository and as a regular Open
Container Initiative (OCI) image released in the
[GitHub container registry](https://github.com/mconfalonieri/external-dns-hetzner-webhook/pkgs/container/external-dns-hetzner-webhook).

The deployment can be performed in every way Kubernetes supports. The examples
provided in this page feature the
[External DNS chart](#using-the-externaldns-chart) and the
[Bitnami chart](#using-the-bitnami-chart).

In both cases the Hetzner API key must be stored in a secret:

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
      tag: "{{ config.extra.image_tag }}"
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

!!! note
    The Bitnami distribution model changed and most features are now paid for.

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
    image: ghcr.io/mconfalonieri/external-dns-hetzner-webhook:{{ config.extra.image_tag }}
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

## Additional notes

You might notice that both examples start the ExternalDNS executables with the
following parameter:

```
--txt-prefix=reg-%{record_type}.
```

The reason behind this is ExternalDNS [issue 6125](https://github.com/kubernetes-sigs/external-dns/issues/6125),
that prevents other types of prefixes to correctly handle the DNS TXT database
for root domain records (@).
