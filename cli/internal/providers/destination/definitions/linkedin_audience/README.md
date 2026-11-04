# Linkedin Audience (`linkedin_audience`)

LinkedIn Audience is an audience destination. RudderStack syncs the rows of a Reverse ETL source into a LinkedIn matched audience, adding and removing members as the rows change.

In a Linkedin Audience destination spec:

- `type: linkedin_audience`
- `definition_version: 1`

> [!NOTE]
> `linkedin_audience` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: linkedin_audience-prod
spec:
  id: linkedin_audience-prod
  display_name: Linkedin Audience Production
  type: linkedin_audience
  definition_version: 1
  enabled: true
  config:
    rudder_account_id: linkedin-audience-account

    connection_mode:
      warehouse: cloud
```

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys and defaults.

### Connection

#### `rudder_account_id` — string, required

The RudderStack account holding the LinkedIn OAuth grant. Create the account in the dashboard and name it here; Rudder CLI does not manage accounts, and a value naming no account in the workspace fails at apply with `Account not found with given id in the workspace`.

This is the whole config surface. Upstream declares no other key: the LinkedIn ad account comes from the linked account, and the fields to sync come from the connection's own identifiers.

### Per-source keys

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach LinkedIn in. LinkedIn Audience accepts only `warehouse: cloud`.

> [!NOTE]
> This destination models no `consent_management` key. Upstream scopes only `connectionMode` to the `warehouse` source type, so it declares no consent surface, and a spec that sets the key fails with `unknown config field`.

## Source types

| Source type | Connection mode |
| :-----| :-----|
| `warehouse` | `cloud` |

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types). LinkedIn Audience accepts no other source type, so Reverse ETL is the only way to reach it.

## Connect a source

A Reverse ETL source connects to this destination and syncs with `sync_behaviour: mirror`:

```yaml
version: rudder/v1
kind: retl-connections
metadata:
  name: retl-connections
spec:
  connections:
    - id: audience-to-linkedin_audience
      source: "#retl-source-sql-model:audience-members"
      destination: "#destination:linkedin_audience-prod"
      config:
        sync_behaviour: mirror
        schedule:
          type: basic
          every_minutes: 60
        identifiers:
          - from: email
            to: user_id
```

Linkedin Audience declares `supportsVisualMapperV2`, so the connection runs the visual mapper v2 flow: the destination names its own field set through its config, an `object` is not allowed, because the destination config carries that field set instead, and `mirror` is the only sync behaviour either side accepts. `mappings` are optional on this flow, not forbidden: add them to fill the declared fields from source columns beyond the identifier, and omit them for the smallest valid connection. Naming an `object` fails, and so does `upsert` or `full`.

Event stream sources are refused by type. Note this is a different destination from `linkedin_ads`, which takes eleven source types and carries its own conversion mapping.

## Secrets

This destination declares no secret keys. The LinkedIn credentials live on the account `rudder_account_id` names, not in `config`.
