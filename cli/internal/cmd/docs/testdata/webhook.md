# webhook (Webhook), definition_version 1, rudder-cli 1.2.3, verified

Narrowed to `javascript` sources, which connect as source token `web`.

Each key goes under `spec.config`. `[]` marks the items of a list. A nested key marked required is required once its parent block or list item is written, and a nested default applies only once its parent block is written. A pattern applies to literal values; a key that takes a template also accepts a `{{ path || fallback }}` template in its place.

| key | type | required | default | allowed values or pattern | secret |
| --- | --- | --- | --- | --- | --- |
| `webhook_url` | `string` | yes |  | pattern `webhook_url`, or a template |  |
| `webhook_method` | `string` |  | `POST` | `POST`, `PUT`, `PATCH`, `GET`, `DELETE` |  |
| `headers[].from` | `string` |  |  | pattern `single_line_1000`, or a template |  |
| `headers[].to` | `string` |  |  | pattern `single_line_1000`, or a template | yes, `STOREFRONT_WEBHOOK_HEADERS_<N>_TO` |
| `connection_mode` | `map<sourceToken,string>` |  |  | see connection modes |  |
| `consent_management` | `map<sourceToken,array<object>>` |  |  |  |  |

## Patterns

- `single_line_1000`: must be at most 1000 characters and must not contain line breaks. Matches `^(.{0,1000})$`.
- `webhook_url`: must be a public http(s) domain URL and must not use localhost or ngrok. Matches `^(https?://)([a-zA-Z0-9-]{1,63}\.)+[a-zA-Z]{2,}(:(6553[0-5]|655[0-2][0-9]|65[0-4][0-9]{2}|6[0-4][0-9]{3}|[1-5]\d{4}|[1-9]\d{1,3}))?(/.*)?$`, and must not match `^https?://([a-zA-Z0-9-]*\.ngrok\.io|localhost|.*\.localhost)`.

## Connection modes

`connection_mode` needs an entry for each connected source token, set to one of the modes below. A mode with required keys needs those keys in the config.

| source token | modes | required keys |
| --- | --- | --- |
| `web` | `cloud` |  |

## Spec template

```yaml
version: rudder/v1
kind: destination
metadata:
  name: storefront-webhook
spec:
  id: storefront-webhook
  display_name: storefront-webhook
  type: webhook
  definition_version: 1
  enabled: true
  config:
    webhook_url: null
    connection_mode:
      web: cloud
```
