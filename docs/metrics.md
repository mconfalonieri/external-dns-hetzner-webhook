# Exposed metrics

The following metrics related to the API calls towards Hetzner are available
for scraping.

## API calls

| Name                         | Type      | Labels   | Description                                              |
| ---------------------------- | --------- | -------- | -------------------------------------------------------- |
| `successful_api_calls_total` | Counter   | `action` | The number of successful Hetzner API calls               |
| `failed_api_calls_total`     | Counter   | `action` | The number of Hetzner API calls that returned an error   |
| `api_delay_hist`             | Histogram | `action` | Histogram of the delay (ms) when calling the Hetzner API |

## Zones and records

| Name                         | Type      | Labels   | Description                                              |
| ---------------------------- | --------- | -------- | -------------------------------------------------------- |
| `filtered_out_zones`         | Gauge     | _none_   | The number of zones excluded by the domain filter        |
| `skipped_records`            | Gauge     | `zone`   | The number of skipped records per domain                 |

## Rate limit metrics

| Name                      | Type      | Labels   | Description                                         |
| ------------------------- | --------- | -------- | --------------------------------------------------- |
| `ratelimit_limit`         | Gauge     | _none_   | Total API calls that can be performed in a hour     |
| `ratelimit_remaining`     | Gauge     | _none_   | Remaining API calls until the next rate limit reset |
| `ratelimit_reset_seconds` | Gauge     | _none_   | UNIX timestamp for the next rate limit reset        |

The label `action` can assume one of the following values, depending on the
Hetzner API endpoint called.

The actions supported by the regular provider are:

- `get_zones`
- `get_rrsets`
- `create_rrset`
- `update_rrset_ttl`
- `update_rrset_records`
- `update_rrset` (this is the method used to update labels)
- `delete_rrset`

In case `BULK_MODE` is set to true, the following actions will be used instead:

- `get_zones`
- `get_rrsets`
- `import_zonefile`
- `export_zonefile`

The label `zone` can assume one of the zone names as its value.
