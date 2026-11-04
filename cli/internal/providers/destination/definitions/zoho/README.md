# ZOHO (`zoho`)

Zoho is an audience destination. RudderStack syncs the rows of a Reverse ETL source into a Zoho CRM or Zoho Marketing Automation list, adding and removing records as the rows change.

In a Zoho destination spec:

- `type: zoho`
- `definition_version: 1`

> [!NOTE]
> `zoho` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: zoho-prod
spec:
  id: zoho-prod
  display_name: Zoho Production
  type: zoho
  definition_version: 1
  enabled: true
  config:
    rudder_account_id: zoho-account

    connection_mode:
      warehouse: cloud
```

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys and defaults.

### Connection

#### `rudder_account_id` — string, required

The RudderStack account holding the Zoho OAuth grant. Create the account in the dashboard and name it here; Rudder CLI does not manage accounts, and a value naming no account in the workspace fails at apply with `Account not found with given id in the workspace`.

This is the whole non-per-source config surface. Everything else about the sync — the Zoho module and the fields — comes from the linked account and the connection's own mappings.

> [!NOTE]
> Upstream names `rudderAccountId` in its `required` list but declares no property for it, so there is no length or shape constraint to enforce. That is an upstream gap rather than a deliberately open type.

### Per-source keys

Both keys are objects keyed by the local source type.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Zoho in. Zoho accepts only `warehouse: cloud`.

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

| Source type | Connection mode |
| :-----| :-----|
| `warehouse` | `cloud` |

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types). Zoho accepts no other source type, so Reverse ETL is the only way to reach it.

## Connect a source

A Reverse ETL source connects to this destination and syncs with `sync_behaviour: mirror`:

```yaml
version: rudder/v1
kind: retl-connections
metadata:
  name: retl-connections
spec:
  connections:
    - id: audience-to-zoho
      source: "#retl-source-sql-model:audience-members"
      destination: "#destination:zoho-prod"
      config:
        sync_behaviour: mirror
        schedule:
          type: basic
          every_minutes: 60
        identifiers:
          - from: email
            to: user_id
```

Zoho declares `supportsVisualMapperV2`, so the connection runs the visual mapper v2 flow: the destination names its own field set through its config, an `object` is not allowed, because the destination config carries that field set instead, and `mirror` is the only sync behaviour either side accepts. `mappings` are optional on this flow, not forbidden: add them to fill the declared fields from source columns beyond the identifier, and omit them for the smallest valid connection. Naming an `object` fails, and so does `upsert` or `full`.

Event stream sources are refused by type, since `warehouse` is the only source type this destination accepts.

## Secrets

This destination declares no secret keys. The Zoho credentials live on the account `rudder_account_id` names, not in `config`.
