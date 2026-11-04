# Intercom V2 (`intercom_v2`)

Intercom V2 is the account-linked rewrite of the Intercom destination. It takes event stream sources like its predecessor, and also syncs the rows of a Reverse ETL source into Intercom contacts and companies.

In an Intercom V2 destination spec:

- `type: intercom_v2`
- `definition_version: 1`

> [!NOTE]
> `intercom_v2` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: intercom-v2-prod
spec:
  id: intercom-v2-prod
  display_name: Intercom V2 Production
  type: intercom_v2
  definition_version: 1
  enabled: true
  config:
    rudder_account_id: intercom-v2-account
    api_server: US
    send_anonymous_id: false

    connection_mode:
      warehouse: cloud
```

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys and defaults.

### Connection

#### `rudder_account_id` — string, required

The RudderStack account holding the Intercom access token. Create the account in the dashboard and name it here; Rudder CLI does not manage accounts, and a value naming no account in the workspace fails at apply with `Account not found with given id in the workspace`.

Unlike `intercom`, this destination takes no API key or app ID in `config` — the linked account carries them.

#### `api_server` — string, required

Intercom regional API host your workspace lives in: `US`, `Europe` or `Australia`.

- The dashboard defaults this to `US`, but Rudder CLI requires it explicitly, because a default cannot sit on a required key.
- A plain enum. A `{{ .VAR }}` reference from a variable file is resolved before validation, so it is accepted as long as it resolves to one of the three values. What upstream does not allow here is a destination-side `{{ path || fallback }}` template, which this field has no alternative for, so such a literal is rejected.

### Behaviour

#### `send_anonymous_id` — boolean, default `false`

Whether RudderStack sends `anonymousId` as the external ID for a user it cannot identify. With it off, unidentified users are dropped rather than creating anonymous Intercom contacts.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types).

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Intercom in. Intercom V2 accepts `cloud` for every source type it supports: unlike `intercom`, it has no device mode, so the SDKs do not load an Intercom bundle.

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

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
| `warehouse` | `cloud` |

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types). Upstream also lists `amp`, which Rudder CLI never declares because it cannot produce that token.

## Connect a source

Event stream sources connect by source type, as they do for any destination. A Reverse ETL source connects and syncs with `sync_behaviour: mirror`:

```yaml
version: rudder/v1
kind: retl-connections
metadata:
  name: retl-connections
spec:
  connections:
    - id: contacts-to-intercom
      source: "#retl-source-sql-model:audience-members"
      destination: "#destination:intercom-v2-prod"
      config:
        sync_behaviour: mirror
        schedule:
          type: basic
          every_minutes: 60
        identifiers:
          - from: email
            to: user_id
```

Intercom V2 declares `supportsVisualMapperV2`, so the connection runs the visual mapper v2 flow: the destination names its own field set through its config, an `object` is not allowed, because the destination config carries that field set instead, and `mirror` is the only sync behaviour either side accepts. `mappings` are optional on this flow, not forbidden: add them to fill the declared fields from source columns beyond the identifier, and omit them for the smallest valid connection. Naming an `object` fails, and so does `upsert` or `full`.

`mirror` being the only accepted behaviour applies to Reverse ETL connections alone. Event stream connections never negotiate a sync behaviour.

## Secrets

This destination declares no secret keys. The Intercom access token lives on the account `rudder_account_id` names, not in `config`.
