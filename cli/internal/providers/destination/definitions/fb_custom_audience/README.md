# Facebook Custom Audience (`fb_custom_audience`)

Facebook Custom Audience is an audience destination. RudderStack syncs the rows of a Reverse ETL source into a Meta custom audience, adding and removing members as the rows change.

In a Facebook Custom Audience destination spec:

- `type: fb_custom_audience`
- `definition_version: 1`

> [!NOTE]
> `fb_custom_audience` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: fb-custom-audience-prod
spec:
  id: fb-custom-audience-prod
  display_name: Facebook Custom Audience Production
  type: fb_custom_audience
  definition_version: 1
  enabled: true
  config:
    rudder_account_id: fb-audience-account
    ad_account_id: act_1234567890
    user_schema:
      - EMAIL
      - PHONE
      - FN
      - LN
    is_hash_required: true
    type: FILE_IMPORTED
    sub_type: MULTI_HASHES

    connection_mode:
      warehouse: cloud
```

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys and defaults.

### Connection

#### `rudder_account_id` — string, required

The RudderStack account holding the Meta access token. Create the account in the dashboard and name it here; Rudder CLI does not manage accounts, and a value naming no account in the workspace fails at apply with `Account not found with given id in the workspace`.

- At most 100 characters, and must not contain line breaks.
- Upstream declares the bare constraint, so a `{{ .VAR }}` reference is measured as text rather than resolved against it.

#### `ad_account_id` — string

Meta ad account the audience belongs to, such as `act_1234567890`. **Required to connect a Reverse ETL source** — see [Connect a source](#connect-a-source).

#### `audience_id` — string

Existing Meta custom audience to sync into. Required to connect a cloud-mode event stream source; a Reverse ETL connection does not use it.

#### `access_token`, `app_secret` — string, secret

Meta credentials from before account linking. `rudder_account_id` supplies them now, so leave both out unless an existing destination carries them. Both are [secret keys](#secrets).

### Audience shape

#### `user_schema` — list of strings

Identifiers Meta matches members on. Accepted members: `EMAIL`, `PHONE`, `GEN`, `MADID`, `EXTERN_ID`, `DOBY`, `DOBM`, `DOBD`, `LN`, `FN`, `FI`, `CT`, `ST`, `ZIP`, `COUNTRY`.

Meta ORs across identifiers, so more mappings raise the match rate. `FN`, `LN`, `CT`, `ST`, `ZIP` and `COUNTRY` only contribute when sent together.

- The dashboard defaults this to `[EMAIL]`. Rudder CLI does not apply that default, because it only applies defaults to single values, so set the key explicitly or the destination stores an empty list.

#### `is_hash_required` — boolean, default `true`

Whether RudderStack SHA256-hashes the identifier values before sending them. Leave it on and send raw warehouse values; pre-hashed input double-hashes and matches nothing.

#### `is_raw` — boolean, default `false`

#### `disable_format` — boolean, default `false`

Turns off the normalization RudderStack applies before hashing (email lowercasing, phone digit stripping, ZIP trimming).

#### `type` — string, default `NA`

Meta audience type. One of `UNKNOWN`, `FILE_IMPORTED`, `EVENT_BASED`, `SEED_BASED`, `THIRD_PARTY_IMPORTED`, `COPY_PASTE`, `CONTACT_IMPORTER`, `HOUSEHOLD_AUDIENCE`, `NA`. A warehouse sync is `FILE_IMPORTED`.

#### `sub_type` — string, default `NA`

Meta audience subtype, one of the 50 values Meta's API accepts. A warehouse sync of hashed identifiers is `MULTI_HASHES`.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types).

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach the destination in. Facebook Custom Audience accepts `cloud` for every source type it supports.

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

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
    - id: audience-to-fb
      source: "#retl-source-sql-model:audience-members"
      destination: "#destination:fb-custom-audience-prod"
      config:
        sync_behaviour: mirror
        schedule:
          type: basic
          every_minutes: 60
        identifiers:
          - from: email
            to: user_id
```

Facebook Custom Audience declares `supportsVisualMapperV2`, so the connection runs the visual mapper v2 flow: the destination names its own field set through `user_schema`, the connection carries no `object` and no `mappings`, and `mirror` is the only sync behaviour either side accepts. Naming an `object` fails with `'object' is not allowed`, and `upsert` or `full` fails with `'sync_behaviour' must be one of [mirror] \u2026 on the visual_mapper_v2 flow`.

The connection also needs `ad_account_id` on the destination, the key the `warehouse` source type requires to connect:

```text
destination 'fb-custom-audience-prod' config is missing fields required to connect a 'warehouse' source: ad_account_id
```

## Secrets

`access_token` and `app_secret` are the secret keys. Write each as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes.
