# Zendesk (`zendesk`)

Zendesk is a customer support destination. RudderStack creates and updates Zendesk users from `identify` events and organizations from `group` events, and records `track` events as events on the user's profile, from its servers.

In a Zendesk destination spec:

- `type: zendesk`
- `definition_version: 1`

> [!NOTE]
> `zendesk` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: zendesk-prod
spec:
  id: zendesk-prod
  display_name: Zendesk Production
  type: zendesk
  definition_version: 1
  enabled: true
  config:
    email: support-admin@acme.com
    api_token: "{{ .ZENDESK_API_TOKEN }}"
    domain: acme
    source_name: Acme Web

    create_users_as_verified: false
    send_group_calls_without_user_id: false
    remove_users_from_organization: false
    search_by_external_id: true

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example makes the email in each `identify` event the user's primary email in Zendesk — which is what `search_by_external_id` does, despite its name. See [Users and organizations](#users-and-organizations).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Zendesk accepts only `identify`, `group`, and `track` events. `identify` and `track` events need a `userId`, and a `track` event also needs the user's `email` trait.

### Connection

#### `email` — string, required

Email address of the Zendesk agent or admin RudderStack authenticates as, together with `api_token`.

- At most 100 characters, and must not contain line breaks. The email format isn't checked.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `api_token` — string, required, secret

Zendesk API token, used with `email` for API token authentication.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- A spec that omits this key fails validation with `'api_token' is required`.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `domain` — string, required

Your Zendesk subdomain, without `.zendesk.com` — `acme` for `acme.zendesk.com`. The dashboard calls it **Zendesk Subdomain**.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- Rudder CLI doesn't check the format, so a full host such as `acme.zendesk.com` passes `validate`, and then breaks delivery.

#### `source_name` — string

Source RudderStack records on the Zendesk events it creates from `track` calls. When omitted, RudderStack uses `Rudder`.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- `zendesk` is reserved, in any capitalization. Rudder CLI accepts it, but RudderStack then fails every `track` event.

### Users and organizations

#### `create_users_as_verified` — boolean, default `false`

Create users as verified, so Zendesk skips its email verification for them.

#### `send_group_calls_without_user_id` — boolean, default `false`

Let a `group` event without a `userId` create or update the organization on its own. When `false`, a `group` event needs a `userId`, and RudderStack adds that user to the organization.

#### `remove_users_from_organization` — boolean, default `false`

Let an `identify` event remove the user from an organization. The event's traits name the organization's Zendesk ID in `company.id`, and set `company.remove` to `true`.

#### `search_by_external_id` — boolean, default `false`

Make the email in an `identify` event the user's primary email in Zendesk, replacing the previous one. When `false`, RudderStack adds that email as a secondary email if the user already has a primary one.

> [!WARNING]
> Despite its name, this key doesn't change how RudderStack finds users. It's the setting the dashboard calls **Update user's primary email**.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Zendesk in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Zendesk accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud` |
| `ios_swift` | `cloud` |
| `web` | `cloud` |
| `unity` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |
| `cloud` | `cloud` |

Every source type is `cloud` only — events reach Zendesk from RudderStack's servers, never in device mode.

The dashboard also accepts `warehouse` for Zendesk, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'zendesk-prod' (type 'zendesk') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'zendesk-prod' config has no 'connection_mode' entry for source type 'web'
```

Zendesk needs no additional config keys to connect a source of any type.

## Secrets

`api_token` is the only secret key. Write it as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  api_token: "{{ .ZENDESK_API_TOKEN }}"
```

`email` isn't a secret — it doesn't authenticate without the token.
