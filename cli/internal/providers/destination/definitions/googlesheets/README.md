# Google Sheets (`googlesheets`)

Google Sheets is a streaming destination. RudderStack appends each event as a row to one sheet of a Google Sheets spreadsheet, writing the event fields you map into columns.

In a Google Sheets destination spec:

- `type: googlesheets`
- `definition_version: 1`

> [!NOTE]
> `googlesheets` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: google-sheets-prod
spec:
  id: google-sheets-prod
  display_name: Google Sheets Production
  type: googlesheets
  definition_version: 1
  enabled: true
  config:
    credentials: '{{ .GOOGLE_SHEETS_CREDENTIALS }}'
    sheet_id: 1kZ8rTq3mVn5Lp0XwYb7Hc2Jd9Fg4Sa6Ue1Ri8Ot3Qy
    sheet_name: Events

    event_key_map:
      - from: event
        to: Event
      - from: userId
        to: User ID
      - from: email
        to: Email
      - from: properties.revenue
        to: Revenue

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example writes four columns after `messageId`, which RudderStack always puts first — see [`event_key_map`](#event_key_map--array-of-objects-required). `credentials` is a JSON key, so its reference sits in single quotes — see [Secrets](#secrets).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

### Connection

#### `credentials` — string, required, secret

JSON key of the Google Cloud service account RudderStack writes to the spreadsheet as.

- The service account needs permission to edit the spreadsheet, and the Google Sheets API must be enabled in the service account's Google Cloud project.
- Not validated locally beyond being present.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `sheet_id` — string, required

ID of the spreadsheet: the part of its URL between `/spreadsheets/d/` and the next `/`.

> [!WARNING]
> Despite its name, `sheet_id` identifies the whole spreadsheet, not one sheet in it — it isn't the numeric `gid` in the URL. `sheet_name` picks the sheet.

- Not validated locally beyond being present.

#### `sheet_name` — string, required

Name of the sheet RudderStack writes to, as shown on its tab — for example `Sheet1`. The sheet must already exist in the spreadsheet.

- Not validated locally beyond being present.

### Column mapping

#### `event_key_map` — array of objects, required

Columns RudderStack writes for each event, in order, after a first `messageId` column.

- `from` — the event field to read: a top-level field such as `event`, or a dotted path such as `properties.revenue` or `context.app.build`. RudderStack reads it from the root of the event first and, when nothing's there, looks it up in `properties`, then `traits`, then `context.traits` — so a bare `email` finds `properties.email`, `traits.email`, or `context.traits.email`, in that order.
- `to` — the column header.
- A field the event doesn't carry leaves its cell empty, and so does a `0` or `false` value.
- RudderStack writes `messageId` and the `to` headers into the sheet's first row, overwriting whatever is there, and appends each event as a new row below.

```yaml
event_key_map:
  - from: properties.revenue
    to: Revenue
  - from: context.app.build
    to: App Build
```

Rudder CLI requires the key but accepts an empty list, and validates neither field of an entry. With an empty list, each row carries only `messageId`.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Google Sheets in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Google Sheets accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud` |
| `ios_swift` | `cloud` |
| `web` | `cloud` |
| `unity` | `cloud` |
| `cloud` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |

Every source type is `cloud` only — RudderStack writes to the spreadsheet from its servers, never in device mode.

The dashboard also accepts `warehouse` for Google Sheets, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'google-sheets-prod' (type 'googlesheets') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'google-sheets-prod' config has no 'connection_mode' entry for source type 'web'
```

Google Sheets needs no additional config keys to connect a source of any type.

## Secrets

`credentials` is the only secret key. Write it as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  credentials: '{{ .GOOGLE_SHEETS_CREDENTIALS }}'
```

`credentials` is JSON, so write its reference in single quotes, as above — see [Secrets](../README.md#secrets). Export the key file as is:

```bash
export RUDDER_GOOGLE_SHEETS_CREDENTIALS="$(cat service-account.json)"
```
