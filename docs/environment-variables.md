# Environment variables

The following environment variables can be used for configuring the application.

## Hetzner DNS API calls configuration

These variables control the behavior of the webhook when interacting with
Hetzner DNS API.

| Variable        | Description                            | Notes                      |
| --------------- | -------------------------------------- | -------------------------- |
| HETZNER_API_KEY | Hetzner API token                      | Mandatory                  |
| BATCH_SIZE      | Number of zones per call               | Default: `100`, max: `100` |
| USE_CLOUD_API   | Use the new cloud API                  | Default: `false`           |
| SLASH_ESC_SEQ   | Escape sequence for label annotations  | Default: `--slash--`       |
| MAX_FAIL_COUNT  | Number of failed calls before shutdown | Default: `-1` (disabled)   |
| ZONE_CACHE_TTL  | TTL for the zone cache in seconds      | Default: `0` (disabled)    |
| BULK_MODE       | Enables bulk mode                      | Default: `false`           |

> [!IMPORTANT]
> Please notice that when **USE_CLOUD_API** is set to `true`, the token stored 
> in **HETZNER_API_KEY** must be a Hetzner Cloud token, NOT the classic DNS one.

## Test and debug

These environment variables are useful for testing and debugging purposes.

| Variable        | Description                      | Notes            |
| --------------- | -------------------------------- | ---------------- |
| DRY_RUN         | If set, changes won't be applied | Default: `false` |
| HETZNER_DEBUG   | Enables debugging messages       | Default: `false` |

## Socket configuration

These variables control the sockets that this application listens to.

| Variable        | Description                      | Notes                |
| --------------- | -------------------------------- | -------------------- |
| WEBHOOK_HOST    | Webhook hostname or IP address   | Default: `localhost` |
| WEBHOOK_PORT    | Webhook port                     | Default: `8888`      |
| METRICS_HOST    | Metrics hostname                 | Default: `0.0.0.0`   |
| METRICS_PORT    | Metrics port                     | Default: `8080`      |
| READ_TIMEOUT    | Sockets' read timeout in ms      | Default: `60000`     |
| WRITE_TIMEOUT   | Sockets' write timeout in ms     | Default: `60000`     |

Please notice that the following variables were **deprecated**:

| Variable    | Description                            |
| ----------- | -------------------------------------- |
| HEALTH_HOST | Metrics hostname (deprecated)          |
| HEALTH_PORT | Metrics port (deprecated)              |
| DEFAULT_TTL | The default TTL is taken from the zone |


## Domain filtering

Additional environment variables for domain filtering. When used, this webhook
will be able to work only on domains (zones) matching the filter.

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
