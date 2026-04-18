# Upgrading from previous versions

## 0.11.x to 0.12.x

No changes to the configuration. Added [rate limit metrics](#exposed-metrics)
for the Cloud API provider.

## 0.10.x to 0.11.x

The configuration is compatible with previous versions; however the
**DEFAULT_TTL** parameter was removed and this environment variable will
therefore not affect the configuration. The default TTL will be the one defined
in the zone.

A **BULK_MODE** parameter was added. When set to `true`, the webhook will
export and manipulate the zonefiles instead of the single recordsets. This will
reduce the API calls when updating zones with lots of changes and a relatively
long interval.

> [!WARNING]
> The bulk mode is experimental and comes with some limitations. Please read
> the [Bulk mode](#bulk-mode) section before activating it.
>

## 0.9.x to 0.10.x

The configuration is fully compatible. There is a new configuration parameter
**ZONE_CACHE_TTL** that controls the TTL of the newly implemented zones
cache, that is aimed to reduce API calls. The parameter is expressed in
seconds.

## 0.8.x to 0.9.x

The configuration is fully compatible. A new configuration parameter
**MAX_FAIL_COUNT** was added to control the webhook behavior in case of
repeated failed attempts to retrieve the records. If this parameter is set to
a value strictly greater than zero, the webhook will shut down after the
configured number of attempts. The default is `-1` (shutdown disabled).

MX records are now supported for both the Cloud API (since **v0.9.1**) and the
Legacy DNS API (since **v0.9.2**).

## 0.7.x to 0.8.x

The configuration is still compatible, however some changes were introduced that
might be worth to check out:

- the new Cloud API is now supported through the **USE_CLOUD_API** environment
  variable and using a Cloud API token for **HETZNER_API_TOKEN**;
- [Hetzner labels](#hetzner-labels) are available when the new Cloud API is in use;
- The minimum ExternalDNS version is now **0.19.0** as the label system and the
  Cloud API are untested with previous versions.

Please notice that the Cloud API requires migrating the DNS zones to the new
DNS tab in Hetzner console.

## 0.6.x to 0.7.x

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
