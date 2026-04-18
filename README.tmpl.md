# ExternalDNS - UNOFFICIAL Hetzner Webhook

> [!IMPORTANT]
> Support for the legacy DNS system is going to be discontinued by Hetzner in
> May 2026. For this reason the legacy DNS provider will be deleted in version
> **v1.0.0**, which will be released in June, and only the new Cloud provider
> will be available.
>
> For the time being no new features will be added to the legacy DNS driver and
> only important bugfixes will be backported.

> [!NOTE]
> The latest version is **{{ .Version }}**.

[ExternalDNS](https://github.com/kubernetes-sigs/external-dns) is a Kubernetes
add-on for automatically DNS records for Kubernetes services using different
providers. By default, Kubernetes manages DNS records internally, but
ExternalDNS takes this functionality a step further by delegating the management
of DNS records to an external DNS provider such as this one. This webhook allows
you to manage your Hetzner domains inside your kubernetes cluster.

This webhook supports both the old DNS API and the new Cloud DNS interface.

> [!TIP]
> If you are upgrading from previous versions please read the
> [Upgrading from previous versions](https://mconfalonieri.github.io/external-dns-hetzner-webhook/{{ .Version }}/upgrading/)
> section.

## 📚 Documentation

Please check the [documentation website](https://mconfalonieri.github.io/external-dns-hetzner-webhook/)
for further information.

## 🚀 Quickstart

This quickstart assumes that you are using the new Cloud API for new
installations.

### 1. Create a Hetzner API Token

Generate a Read/Write API token in your [Hetzner Console](https://console.hetzner.com/).

### 2. Create a secret with your API token

Substitute `<CLOUD_API_TOKEN>` with your token:

```yaml
kubectl create secret generic hetzner-credentials --from-literal=api-key='<CLOUD_API_TOKEN>' -n external-dns
```

### 3. Deploy ExternalDNS with the webhook provider

The simplest way is using Helm.

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
# install external-dns with Helm
helm install external-dns-hetzner external-dns/external-dns -f external-dns-hetzner-values.yaml -n external-dns
```

## ⚖️ License

This project is licensed under the Apache License, version 2.0 - see the
[LICENSE](./LICENSE) file for details.

## 👥 Credits

This Webhook was forked and modified from the [IONOS Webhook](https://github.com/ionos-cloud/external-dns-ionos-webhook)
to work with Hetzner. It also contains huge parts from
[DrBu7cher's Hetzner provider](https://github.com/DrBu7cher/external-dns/tree/readding_hcloud_provider).

### Contributors

| Name                                         | Contribution                  |
| -------------------------------------------- | ----------------------------- |
| [DerQue](https://github.com/DerQue)          | local CNAME fix               |
| [sschaeffner](https://github.com/sschaeffner)| build configuration for arm64 |
| [sgaluza](https://github.com/sgaluza)        | support for MX records        |
