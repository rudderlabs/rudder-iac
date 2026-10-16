# Customer.io Audience (`customerio_audience`)

Customer.io Audience is a Reverse ETL destination. RudderStack syncs rows from a warehouse into a Customer.io manual segment, adding people to the segment and removing them as the rows change.

In a Customer.io Audience destination spec:

- `type: customerio_audience`
- `definition_version: 1`

> [!NOTE]
> `customerio_audience` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: customerio-audience-prod
spec:
  id: customerio-audience-prod
  display_name: Customer.io Audience Production
  type: customerio_audience
  definition_version: 1
  enabled: true
  config:
    site_id: 88f02580a3b9c4d7cf18
    api_key: "{{ .CUSTOMERIO_TRACK_API_KEY }}"
    app_api_key: "{{ .CUSTOMERIO_APP_API_KEY }}"
    region: US

    connection_mode:
      warehouse: cloud
```

The above example uses `{{ .VAR }}` references for both API keys — see [Secrets](#secrets). Rudder CLI creates and updates the destination, but it can't connect any source to it — see [Connect a source](#connect-a-source).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Customer.io Audience accepts only `record` events — the rows a Reverse ETL sync produces — and runs only in `cloud` mode.

### Connection

#### `site_id` — string, required

Customer.io site ID, listed with its Track API key under **API Credentials** in your Customer.io account settings. RudderStack pairs it with `api_key` to authenticate to Customer.io's Track API.

- At most 100 characters, and must not contain line breaks.
- Templates aren't accepted in place of a literal; a template is measured as text against the same limit.

#### `api_key` — string, required, secret

Track API key that belongs to `site_id`.

- At most 100 characters, and must not contain line breaks, on the same terms as `site_id`.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `app_api_key` — string, required, secret

Customer.io App API key, from the **App API Keys** list under **API Credentials**. It's a different credential from the Track API key in `api_key`.

- At most 100 characters, and must not contain line breaks, on the same terms as `site_id`.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `region` — string, required

Customer.io data center your account lives in: `US` or `EU`.

- The dashboard defaults this field to `US`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation with `'region' is required`.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Customer.io Audience in. Customer.io Audience accepts only `warehouse: cloud`.

- Rudder CLI can't connect a source to this destination, so no check ever asks for the entry — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  warehouse: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Customer.io Audience accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `warehouse` | `cloud` |

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types). Customer.io Audience accepts no other source type, even in the dashboard.

## Connect a source

> [!WARNING]
> Rudder CLI can't connect any source to Customer.io Audience. It refuses every event stream source by type, and every Reverse ETL connection by flow. Connect a Reverse ETL source to this destination in the dashboard instead.

**Event stream sources aren't supported.** Customer.io Audience accepts only `warehouse`, the type a Reverse ETL source resolves to. Every event stream source resolves to a type it doesn't accept — a JavaScript source to `web`, and webhook and server-side SDK sources to `cloud` — and reports, for example:

```text
destination 'customerio-audience-prod' (type 'customerio_audience') does not support source 'my-js-source':
source type 'web' is not among supported source types: warehouse
```

**Reverse ETL connections are refused outright.** A Reverse ETL source resolves to `warehouse`, which the definition accepts, but Customer.io Audience syncs segments through a destination-specific Reverse ETL flow that Rudder CLI doesn't support:

```text
destination api type "CUSTOMERIO_AUDIENCE" uses a destination-specific rETL flow, which is not supported
```

Neither refusal depends on the destination config, so no `config` change clears it.

`rudder-cli import` skips a Customer.io Audience connection made in the dashboard for the same reason, and leaves it in the workspace unchanged.

## Secrets

`api_key` and `app_api_key` are the secret keys. Write each as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  api_key: "{{ .CUSTOMERIO_TRACK_API_KEY }}"
  app_api_key: "{{ .CUSTOMERIO_APP_API_KEY }}"
```

`site_id` isn't a secret, though RudderStack sends it alongside `api_key` to authenticate.
