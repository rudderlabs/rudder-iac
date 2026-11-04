# Braze Audiences (`braze_audience`)

Braze Audiences is an audience destination. RudderStack syncs the rows of a Reverse ETL source into a Braze subscription group or user cohort, adding and removing members as the rows change.

In a Braze Audiences destination spec:

- `type: braze_audience`
- `definition_version: 1`

> [!NOTE]
> `braze_audience` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: braze_audience-prod
spec:
  id: braze_audience-prod
  display_name: Braze Audiences Production
  type: braze_audience
  definition_version: 1
  enabled: true
  config:
    rudder_account_id: braze-audience-account

    connection_mode:
      warehouse: cloud
```

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys and defaults.

### Connection

#### `rudder_account_id` — string, required

The RudderStack account holding the Braze REST API key. Create the account in the dashboard and name it here; Rudder CLI does not manage accounts, and a value naming no account in the workspace fails at apply with `Account not found with given id in the workspace`.

- At most 100 characters, and must not contain line breaks.
- Upstream declares the bare constraint, so a `{{ .VAR }}` reference is measured as text rather than resolved against it.

#### `rest_api_key` — string, secret

#### `data_center` — string

Braze settings from before account linking. `rudder_account_id` supplies them now, so leave both out unless an existing destination carries them. Upstream declares no constraint on either, so Rudder CLI validates nothing beyond their type and a wrong value is reported by Braze at apply. `rest_api_key` is a [secret key](#secrets).

### Per-source keys

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Braze in. Braze Audiences accepts only `warehouse: cloud`.

> [!NOTE]
> This destination models no `consent_management` key. Upstream scopes only `connectionMode` to the `warehouse` source type, so it declares no consent surface, and a spec that sets the key fails with `unknown config field`.

## Source types

| Source type | Connection mode |
| :-----| :-----|
| `warehouse` | `cloud` |

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types). Braze Audiences accepts no other source type, so Reverse ETL is the only way to reach it.

## Connect a source

A Reverse ETL source connects to this destination and syncs with `sync_behaviour: mirror`:

```yaml
version: rudder/v1
kind: retl-connections
metadata:
  name: retl-connections
spec:
  connections:
    - id: audience-to-braze_audience
      source: "#retl-source-sql-model:audience-members"
      destination: "#destination:braze_audience-prod"
      config:
        sync_behaviour: mirror
        schedule:
          type: basic
          every_minutes: 60
        identifiers:
          - from: email
            to: user_id
```

Braze Audiences declares `supportsVisualMapperV2`, so the connection runs the visual mapper v2 flow: the destination names its own field set through its config, the connection carries no `object` and no `mappings`, and `mirror` is the only sync behaviour either side accepts. Naming an `object` fails, and so does `upsert` or `full`.

Event stream sources are refused by type, since `warehouse` is the only source type this destination accepts. Note this is a different destination from `braze`, which takes event stream sources and is verified.

## Secrets

`rest_api_key` is the only secret key. Write it as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the value and what `import` writes.
