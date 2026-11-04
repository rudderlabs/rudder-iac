# Google Ads Remarketing Lists (`google_adwords_remarketing_lists`)

Google Ads Remarketing Lists (Customer Match) is an audience destination. RudderStack syncs the rows of a Reverse ETL source into a Google Ads customer match list, adding and removing members as the rows change.

In a Google Ads Remarketing Lists destination spec:

- `type: google_adwords_remarketing_lists`
- `definition_version: 1`

> [!NOTE]
> `google_adwords_remarketing_lists` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: gads-remarketing-prod
spec:
  id: gads-remarketing-prod
  display_name: Google Ads Remarketing Production
  type: google_adwords_remarketing_lists
  definition_version: 1
  enabled: true
  config:
    rudder_account_id: gads-remarketing-account
    customer_id: "1234567890"
    sub_account: false
    user_schema:
      - email
      - phone
    is_hash_required: true
    type_of_list: General

    connection_mode:
      warehouse: cloud
```

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys and defaults.

### Connection

#### `rudder_account_id` — string, required

The RudderStack account holding the Google Ads OAuth grant. Create the account in the dashboard and name it here; Rudder CLI does not manage accounts, and a value naming no account in the workspace fails at apply with `Account not found with given id in the workspace`.

#### `customer_id` — string, required

Google Ads customer ID the list belongs to, digits only without dashes.

- At most 100 characters, and must not contain line breaks. A `{{ .VAR }}` reference is accepted in place of a literal.

#### `sub_account` — boolean, required

Whether `customer_id` names a sub-account reached through a manager account. The dashboard defaults this to `false`, but Rudder CLI requires it explicitly, because the key decides whether `login_customer_id` is needed.

#### `login_customer_id` — string

Manager account ID that authorizes access to `customer_id`. Required when `sub_account` is `true`, and ignored otherwise:

```text
'login_customer_id' is required when 'sub_account' is 'true'
```

#### `list_id` — string

Existing Google Ads user list to sync into. Google Ads mints one when the key is absent.

#### `audience_id` — string

Required to connect a cloud-mode event stream source; a Reverse ETL connection does not use it.

### Audience shape

#### `user_schema` — list of strings

Identifiers Google matches members on: `email`, `phone`, `addressInfo`. Note the lowercase spelling — Google Ads differs from Facebook Custom Audience here.

- Defaults to `[email]`. Rudder CLI applies that default when the spec omits the key, matching what the backend stores, so an omitted key does not diff on later applies. Set it explicitly to match on more than email.

#### `is_hash_required` — boolean, default `true`

Whether RudderStack SHA256-hashes the identifier values before sending them. Leave it on and send raw warehouse values.

#### `type_of_list` — string, default `General`

One of `General`, `userID`, `mobileDeviceID`.

### Consent

#### `user_data_consent` — string, default `UNSPECIFIED`

Consent for using the data, passed through to Google Ads. One of `UNSPECIFIED`, `UNKNOWN`, `GRANTED`, `DENIED`.

#### `personalization_consent` — string, default `UNSPECIFIED`

Consent for ad personalization, on the same four values.

#### `consent_management` — object

Consent provider configuration per source type, keyed by the tokens under [Source types](#source-types). The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

### Per-source keys

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Google Ads in. This destination accepts `cloud` for every source type it supports.

```yaml
connection_mode:
  warehouse: cloud
```

## Source types

| Source type | Connection mode |
| :-----| :-----|
| `cloud` | `cloud` |
| `warehouse` | `cloud` |

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types). Upstream also lists `shopify`, which Rudder CLI never declares because it cannot produce that token.

## Connect a source

A Reverse ETL source connects to this destination and syncs with `sync_behaviour: mirror`:

```yaml
version: rudder/v1
kind: retl-connections
metadata:
  name: retl-connections
spec:
  connections:
    - id: audience-to-gads
      source: "#retl-source-sql-model:audience-members"
      destination: "#destination:gads-remarketing-prod"
      config:
        sync_behaviour: mirror
        schedule:
          type: basic
          every_minutes: 60
        identifiers:
          - from: email
            to: user_id
```

Google Ads Remarketing Lists declares `supportsVisualMapperV2`, so the connection runs the visual mapper v2 flow: the destination names its own field set through `user_schema`, an `object` is not allowed, because the destination config carries that field set instead, and `mirror` is the only sync behaviour either side accepts. `mappings` are optional on this flow, not forbidden: add them to fill the declared fields from source columns beyond the identifier, and omit them for the smallest valid connection.

## Secrets

This destination declares no secret keys. The Google Ads credentials live on the account `rudder_account_id` names, not in `config`.
