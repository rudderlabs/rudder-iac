# LinkedIn Ads (`linkedin_ads`)

LinkedIn Ads is an advertising destination. RudderStack sends mapped `track` events to LinkedIn's Conversions API from its servers, as conversions against the conversion rules in your LinkedIn ad account, authenticating with a LinkedIn account connected through OAuth.

In a LinkedIn Ads destination spec:

- `type: linkedin_ads`
- `definition_version: 1`

> [!NOTE]
> `linkedin_ads` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: linkedin-ads-prod
spec:
  id: linkedin-ads-prod
  display_name: LinkedIn Ads Production
  type: linkedin_ads
  definition_version: 1
  enabled: true
  config:
    rudder_account_id: 2fS9kQwPzX7rTnV4yLbH8mJcD1e
    hash_data: true
    ad_account_id: "508412345"
    deduplication_key: properties.eventId

    conversion_mapping:
      - from: Order Completed
        to: "14825392"
      - from: Demo Requested
        to: "14825417"

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
```

The above example deduplicates conversions on `properties.eventId` rather than the event's `messageId` — see [`deduplication_key`](#deduplication_key--string). RudderStack rejects any `track` event that `conversion_mapping` doesn't name — see [Conversion mapping](#conversion-mapping).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> LinkedIn Ads accepts only `track` events, and runs only in `cloud` mode.

### Connection

#### `rudder_account_id` — string, required

ID of the RudderStack account that holds the OAuth connection to LinkedIn. RudderStack authenticates every request with that account, so the destination config carries no LinkedIn credentials of its own.

> [!NOTE]
> Rudder CLI can't create this account. It's an OAuth connection, made by signing in to LinkedIn from the RudderStack dashboard: create it there, then write its ID here. `rudder-cli workspace accounts list` lists the workspace's accounts with their IDs.

- The LinkedIn user who authorizes the account needs one of these ad account roles: `ACCOUNT_BILLING_ADMIN`, `ACCOUNT_MANAGER`, `CAMPAIGN_MANAGER`, or `CREATIVE_MANAGER`.
- Rudder CLI doesn't check the value — neither that the account exists nor that it's a LinkedIn account.

#### `hash_data` — boolean, required

SHA-256 hash the user's email address before sending it. The dashboard calls this **Hash and Encode Data**.

- Set it to `false` only when your events already carry SHA-256-hashed email addresses: RudderStack labels the email as a SHA-256 hash either way.
- The dashboard defaults this field to `true`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation with `'hash_data' is required`.

#### `ad_account_id` — string

ID of the LinkedIn ad account that owns the conversion rules in `conversion_mapping`. The dashboard calls it **LinkedIn Ad Account Id**.

- RudderStack doesn't send it with events: each conversion names its rule by ID alone.
- At most 100 characters, and must not contain line breaks. Write an all-digit ID as a string.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `deduplication_key` — string

Path, from the root of the event, to the value LinkedIn deduplicates conversions on — for example `properties.eventId`. RudderStack sends the event's `messageId` instead when the key is unset, or when the event has no value at the path.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

### Conversion mapping

#### `conversion_mapping` — array of objects

Maps RudderStack event names to LinkedIn conversion rules. RudderStack rejects a `track` event whose name has no entry here, so map every event you send.

- `from` — RudderStack event name, matched exactly, including case.
- `to` — ID of a conversion rule set up for the Conversions API in LinkedIn Campaign Manager. Write an all-digit ID as a string.
- Each entry needs both fields. Each is at most 100 characters and must not contain line breaks, or a `{{ path || fallback }}` template, which isn't measured against the length limit.
- Map an event name more than once to send it as a conversion for each rule.

```yaml
conversion_mapping:
  - from: Order Completed
    to: "14825392"
  - from: Order Completed
    to: "14825430"
```

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach LinkedIn Ads in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

LinkedIn Ads accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — RudderStack sends conversions from its servers, never in device mode.

The dashboard also accepts `warehouse` for LinkedIn Ads, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'linkedin-ads-prod' (type 'linkedin_ads') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'linkedin-ads-prod' config has no 'connection_mode' entry for source type 'web'
```

LinkedIn Ads needs no additional config keys to connect a source of any type. The OAuth account in `rudder_account_id` isn't checked at `validate` time.

## Secrets

LinkedIn Ads has no secret keys. Rudder CLI treats none of its keys as secret, and the dashboard masks none of them. The LinkedIn credentials stay with the OAuth account that `rudder_account_id` names, so a spec for this destination needs no `{{ .VAR }}` references.
