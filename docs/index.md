# Home

## Introduction

[ExternalDNS](https://github.com/kubernetes-sigs/external-dns) is a Kubernetes
add-on for automatically DNS records for Kubernetes services using different
providers. By default, Kubernetes manages DNS records internally, but
ExternalDNS takes this functionality a step further by delegating the management
of DNS records to an external DNS provider such as this one. This webhook allows
you to manage your Hetzner domains inside your kubernetes cluster.

This webhook supports both the old DNS API and the new Cloud DNS interface.

## Table of contents

- [Requirements](./requirements.md): requirements for using this software.
- [Deployment](./deployment.md): how to deploy this software in Kubernetes.
- [Environment variables](./environment-variables.md): the configuration is
  managed via environment variables.
- [Upgrading from previous versions](./upgrading.md): a version-by-version
  guide on how to upgrade a working installation. It includes some notes on
  each version.
- [Endpoints](./endpoints.md): the endpoints exposed by this webhook.
- [Metrics](./metrics.md): metrics exposed by this webhook.
- [Advanced features](./advanced-features.md): advanced features implemented in
  this webhook:

  - zone cache
  - Hetzner labels
  - bulk mode
