# Iterable Audience (`iterable_audience`)

Iterable Audience is an audience destination. RudderStack syncs the rows of a Reverse ETL source into an Iterable static list, adding and removing members as the rows change.

In an Iterable Audience destination spec:

- `type: iterable_audience`
- `definition_version: 1`

> [!NOTE]
> `iterable_audience` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: iterable-audience-prod
spec:
  id: iterable-audience-prod
  display_name: Iterable Audience Production
  type: iterable_audience
  definition_version: 1
  enabled: true
  config:
    rudder_account_id: iterable-audience-account

    connection_mode:
      warehouse: cloud
```

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys and defaults.

### Connection

#### `rudder_account_id` — string, required

The RudderStack account holding the Iterable API key. Create the account in the dashboard and name it here; Rudder CLI does not manage accounts, and a value naming no account in the workspace fails at apply with `Account not found with given id in the workspace`.

- At most 100 characters, and must not contain line breaks.
- Rudder CLI resolves a `{{ .VAR }}` reference from a variable file before it validates, so the limit applies to the resolved value. Upstream declares the bare constraint with no template alternative, so a destination-side template that survives substitution is judged as an ordinary literal.

#### `api_key` — string, secret

#### `data_center` — string

#### `project_type` — string

Iterable settings from before account linking. `rudder_account_id` supplies them now and the dashboard no longer shows them, so leave all three out unless an existing destination carries them. Upstream declares no constraint on any of them, so Rudder CLI validates nothing beyond their type and a wrong value is reported by Iterable at apply. `api_key` is a [secret key](#secrets).

### Per-source keys

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Iterable in. Iterable Audience accepts only `warehouse: cloud`.

```yaml
connection_mode:
  warehouse: cloud
```

> [!NOTE]
> This destination models no `consent_management` key. Upstream scopes only `connectionMode` to the `warehouse` source type, so it declares no consent surface, and a spec that sets the key fails with `unknown config field`.

## Source types

| Source type | Connection mode |
| :-----| :-----|
| `warehouse` | `cloud` |

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types). Iterable Audience accepts no other source type, even in the dashboard, so Reverse ETL is the only way to reach it.

## Connect a source

A Reverse ETL source connects to this destination and syncs with `sync_behaviour: mirror`:

```yaml
version: rudder/v1
kind: retl-connections
metadata:
  name: retl-connections
spec:
  connections:
    - id: audience-to-iterable
      source: "#retl-source-sql-model:audience-members"
      destination: "#destination:iterable-audience-prod"
      config:
        sync_behaviour: mirror
        schedule:
          type: basic
          every_minutes: 60
        identifiers:
          - from: email
            to: user_id
```

Iterable Audience declares `supportsVisualMapperV2`, so the connection runs the visual mapper v2 flow: the destination names its own field set through its config, an `object` is not allowed, because the destination config carries that field set instead, and `mirror` is the only sync behaviour either side accepts. `mappings` are optional on this flow, not forbidden: add them to fill the declared fields from source columns beyond the identifier, and omit them for the smallest valid connection.

Event stream sources are refused by type, since `warehouse` is the only source type this destination accepts.

## Secrets

`api_key` is the only secret key. Write it as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the value and what `import` writes.
