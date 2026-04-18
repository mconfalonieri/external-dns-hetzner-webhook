# Requirements

This webhook can be used in conjunction with **ExternalDNS v0.19.0 or higher**,
configured for using the webhook interface. Some examples for a working
configuration are shown in the [deployment](./deployment.md) section.

## Legacy DNS API

A [DNS API token](https://docs.hetzner.com/dns-console/dns/general/api-access-token/)
for the account managing your domains is required for this webhook to work
properly when the **USE_CLOUD_API** environment variable is set to `false` or
not set.

## Cloud API

A [Cloud API token](https://docs.hetzner.com/cloud/api/getting-started/generating-api-token/)
is required when the new Cloud API is in use (the **USE_CLOUD_API** environment
variable is set to `true`).
