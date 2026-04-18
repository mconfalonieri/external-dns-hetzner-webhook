# Advanced features

The features that are not required for basic usage, but are needed for certain
environments are described in this section.

## Zone cache

The zone cache is enabled by setting a value for the environment variable
**ZONE_CACHE_TTL** greater than zero. This parameter allows the webhook to
download once the list of zones and keep using it for the given number of
seconds. When set to zero (default value) the zone cache is disabled and the
zones will be reloaded every time the webhook is called by ExternalDNS.

## Hetzner labels

> ![NOTE]
> This feature is available only in the Cloud API.

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

> ![NOTE]
> This feature is available only in the Cloud API.

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
