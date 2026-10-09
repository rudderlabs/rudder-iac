# Google Ads Offline Conversions (`google_adwords_offline_conversions`)

Google Ads Offline Conversions is an advertising destination. RudderStack turns mapped `track` events into click, call, or store conversions and uploads them to the Google Ads API from its servers, authenticating with a Google Ads account connected through OAuth.

In a Google Ads Offline Conversions destination spec:

- `type: google_adwords_offline_conversions`
- `definition_version: 1`

> [!NOTE]
> `google_adwords_offline_conversions` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: google-ads-offline-prod
spec:
  id: google-ads-offline-prod
  display_name: Google Ads Offline Conversions Production
  type: google_adwords_offline_conversions
  definition_version: 1
  enabled: true
  config:
    rudder_account_id: 2fS9kQwPzX7rTnV4yLbH8mJcD1e
    customer_id: "6293829833"
    sub_account: true
    login_customer_id: "4417203958"

    events_to_offline_conversions_type_mapping:
      - from: Order Completed
        to: click
      - from: Store Purchase
        to: store
    events_to_conversions_names_mapping:
      - from: Order Completed
        to: Online Purchase
      - from: Store Purchase
        to: In-Store Purchase
    custom_variables:
      - from: coupon
        to: coupon_code

    user_identifier_source: FIRST_PARTY
    conversion_environment: WEB
    default_user_identifier: email
    hash_user_identifier: true
    validate_only: false

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
```

The above example reaches the customer account through a manager account, so it sets `sub_account` and `login_customer_id` — see [Connection](#connection). An event becomes a conversion only when both event mappings name it — see [Event mapping](#event-mapping).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Google Ads Offline Conversions accepts only `track` events, and runs only in `cloud` mode.

### Connection

#### `rudder_account_id` — string, required

ID of the RudderStack account that holds the OAuth connection to Google Ads. RudderStack authenticates every upload with that account, so the destination config carries no Google credentials of its own.

> [!NOTE]
> Rudder CLI can't create this account. It's an OAuth connection, made by signing in to Google from the RudderStack dashboard: create it there, then write its ID here. `rudder-cli workspace accounts list` lists the workspace's accounts with their IDs.

- The Google user who authorizes the account needs **Standard** or **Admin** access to the Google Ads account.
- Rudder CLI doesn't check the value — neither that the account exists nor that it's a Google Ads account.

#### `customer_id` — string, required

Google Ads customer ID of the account that owns the conversion actions. RudderStack strips hyphens from it, so `629-382-9833` works as well as `6293829833`.

- At most 100 characters, and must not contain line breaks.
- Write it as a string. An unquoted all-digit ID is read as a number and fails validation.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- Under cross-account conversion tracking the conversion actions live on the manager account, so this is the manager's customer ID.

```yaml
customer_id: "6293829833"
```

#### `sub_account` — boolean, default `false`

Whether `customer_id` is a sub-account that the authorizing Google user reaches through a manager account. When `true`, RudderStack names that manager account, from `login_customer_id`, on every request to Google Ads.

#### `login_customer_id` — string, required

Customer ID of the manager account the authorizing user reaches `customer_id` through.

- Required when `sub_account` is `true`. Without it, validation fails with `'login_customer_id' is required when 'sub_account' is true`.
- Ignored when `sub_account` is `false`, though Rudder CLI still accepts and stores it.
- At most 100 characters, and must not contain line breaks. Hyphens are stripped, as for `customer_id`.
- A `{{ path || fallback }}` template is accepted in place of a literal.

### Event mapping

A `track` event becomes a conversion only when its name appears in both `events_to_offline_conversions_type_mapping` and `events_to_conversions_names_mapping`. RudderStack rejects any other `track` event. Event names match case-insensitively.

#### `events_to_offline_conversions_type_mapping` — array of objects

Maps RudderStack event names to the kind of conversion RudderStack uploads them as.

- `from` — RudderStack event name. At most 100 characters, and must not contain line breaks. A `{{ path || fallback }}` template is accepted in place of a literal. Rudder CLI doesn't measure a template against the length limit, but the API does, so a longer one passes `validate` and fails at `apply`.
- `to` — `click`, `call`, or `store`. A `store` conversion is uploaded as store sales data through an offline user data job. Templates aren't accepted here.
- Map an event name more than once to upload it as more than one kind of conversion.

```yaml
events_to_offline_conversions_type_mapping:
  - from: Order Completed
    to: click
  - from: Order Completed
    to: store
```

#### `events_to_conversions_names_mapping` — array of objects

Maps RudderStack event names to Google Ads conversion actions.

- `from` — RudderStack event name.
- `to` — name of the conversion action in Google Ads. RudderStack looks the action up by this name, so it must match the action's name in Google Ads exactly; an event whose action isn't found fails.
- Both fields are at most 100 characters and must not contain line breaks. A `{{ path || fallback }}` template is accepted in place of a literal in either, on the same terms as `from` in `events_to_offline_conversions_type_mapping`.

```yaml
events_to_conversions_names_mapping:
  - from: Order Completed
    to: Online Purchase
```

#### `custom_variables` — array of objects

Maps event properties to Google Ads custom conversion variables, which must already exist in Google Ads.

- `from` — name of a top-level key in the event's `properties`.
- `to` — name of the custom variable in Google Ads.
- Both fields are at most 100 characters and must not contain line breaks. Templates are accepted as for the event mappings.
- Applies to `click` and `call` conversions only. Store conversions don't carry custom variables.

### Conversion settings

#### `user_identifier_source` — string, default `none`

Source of the user identifier attached to a conversion: `UNSPECIFIED`, `UNKNOWN`, `FIRST_PARTY`, or `THIRD_PARTY`. `none` sends no source.

- An event's own `properties.userIdentifierSource` takes precedence.
- Applies to `click` conversions only.

#### `conversion_environment` — string, default `none`

Environment the conversion was recorded in: `UNSPECIFIED`, `UNKNOWN`, `APP`, or `WEB`. `none` sends no environment.

- An event's own `properties.conversionEnvironment` takes precedence.
- Applies to `click` conversions only.

#### `default_user_identifier` — string, default `email`

User identifier RudderStack prefers for `click` and `store` conversions: `email` or `phone`. When an event lacks the preferred one, RudderStack sends another identifier the event carries instead.

#### `hash_user_identifier` — boolean, default `true`

SHA-256 hash user-identifying information before upload: the email address and phone number, plus first name, last name, and street address on `store` conversions. Turn it off only if your events already carry hashed values.

#### `validate_only` — boolean, default `false`

Have Google Ads validate uploads without applying them.

- Applies to `store` conversions only. Click and call conversions are always applied, whatever this key says.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Google Ads Offline Conversions in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Google Ads Offline Conversions accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — RudderStack uploads conversions from its servers, never in device mode.

The dashboard also accepts `warehouse` for Google Ads Offline Conversions, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'google-ads-offline-prod' (type 'google_adwords_offline_conversions') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'google-ads-offline-prod' config has no 'connection_mode' entry for source type 'web'
```

Google Ads Offline Conversions needs no additional config keys to connect a source of any type. The OAuth account in `rudder_account_id` isn't checked at `validate` time.

## Secrets

Google Ads Offline Conversions has no secret keys. Rudder CLI treats none of its keys as secret, and the dashboard masks none of them. The Google credentials stay with the OAuth account that `rudder_account_id` names, so a spec for this destination needs no `{{ .VAR }}` references.
