# Tiktok Audience (`tiktok_audience`)

TikTok Audience is an audience destination. RudderStack syncs the rows of a Reverse ETL source into a TikTok Ads custom audience, adding and removing members as the rows change.

In a TikTok Audience destination spec:

- `type: tiktok_audience`
- `definition_version: 1`

> [!NOTE]
> `tiktok_audience` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: tiktok-audience-prod
spec:
  id: tiktok-audience-prod
  display_name: TikTok Audience Production
  type: tiktok_audience
  definition_version: 1
  enabled: true
  config:
    rudder_account_id: tiktok-audience-account
    advertiser_id: "7000000000000000000"

    connection_mode:
      warehouse: cloud
```

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys and defaults.

### Connection

#### `rudder_account_id` — string, required

The RudderStack account holding the TikTok Business OAuth grant. Create the account in the dashboard and name it here; Rudder CLI does not manage accounts, and a value naming no account in the workspace fails at apply with `Account not found with given id in the workspace`.

#### `advertiser_id` — string

TikTok advertiser the audience belongs to.

#### `ad_account_id` — string

TikTok ad account. Scoped to the `warehouse` source type upstream rather than being a default key, so it only applies to a Reverse ETL connection.

#### `audience_id` — string

Existing TikTok custom audience to sync into. TikTok mints one when the key is absent.

> [!NOTE]
> `rudder_account_id` is the only key upstream constrains. Rudder CLI validates nothing beyond the type of the other three, so a wrong value is reported by TikTok at apply rather than by `validate`. `audience_id` is also the one key that appears in no `destConfig` list upstream — an upstream gap. It stays modelled, because destination update replaces the whole config object and an unmodelled key would be erased on first apply.

### Per-source keys

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach TikTok in. TikTok Audience accepts only `warehouse: cloud`.

> [!NOTE]
> This destination models no `consent_management` key. Upstream scopes only `adAccountId` and `connectionMode` to the `warehouse` source type, so it declares no consent surface, and a spec that sets the key fails with `unknown config field`.

## Source types

| Source type | Connection mode |
| :-----| :-----|
| `warehouse` | `cloud` |

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types). TikTok Audience accepts no other source type, so Reverse ETL is the only way to reach it.

## Connect a source

A Reverse ETL source connects to this destination and syncs with `sync_behaviour: mirror`:

```yaml
version: rudder/v1
kind: retl-connections
metadata:
  name: retl-connections
spec:
  connections:
    - id: audience-to-tiktok
      source: "#retl-source-sql-model:audience-members"
      destination: "#destination:tiktok-audience-prod"
      config:
        sync_behaviour: mirror
        schedule:
          type: basic
          every_minutes: 60
        identifiers:
          - from: email
            to: user_id
```

TikTok Audience declares `supportsVisualMapperV2`, so the connection runs the visual mapper v2 flow: the destination names its own field set through its config, the connection carries no `object` and no `mappings`, and `mirror` is the only sync behaviour either side accepts. Naming an `object` fails, and so does `upsert` or `full`.

Event stream sources are refused by type. Note this is a different destination from `tiktok_ads`, which takes event stream sources and is verified.

## Secrets

This destination declares no secret keys. The TikTok credentials live on the account `rudder_account_id` names, not in `config`.
