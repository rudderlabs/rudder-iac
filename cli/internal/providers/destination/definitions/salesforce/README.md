# Salesforce (`salesforce`)

Salesforce is a CRM destination. RudderStack creates and updates Salesforce leads — or other Salesforce objects — from `identify` events, from its servers.

In a Salesforce destination spec:

- `type: salesforce`
- `definition_version: 1`

> [!NOTE]
> `salesforce` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

> [!WARNING]
> The dashboard marks this destination as deprecated in favor of Salesforce V2, which Rudder CLI doesn't support.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: salesforce-prod
spec:
  id: salesforce-prod
  display_name: Salesforce Production
  type: salesforce
  definition_version: 1
  enabled: true
  config:
    user_name: rudderstack-integration@acme.com
    password: "{{ .SALESFORCE_PASSWORD }}"
    initial_access_token: "{{ .SALESFORCE_SECURITY_TOKEN }}"
    sandbox: false

    map_properties: true
    use_contact_id: false

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
```

The above example signs in to a production org. `initial_access_token` holds the user's security token, not an access token — see [Authentication](#authentication).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Salesforce accepts only `identify` events. RudderStack doesn't deliver any other event type to it.

### Authentication

RudderStack signs in as a Salesforce user through the OAuth 2.0 username-password flow, which your org must allow. It sends `password` with `initial_access_token` appended, as Salesforce expects for API sign-ins.

#### `user_name` — string, required

Username of the Salesforce user RudderStack signs in as.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `password` — string, required, secret

Password of that user.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `initial_access_token` — string, required, secret

The user's Salesforce security token.

> [!WARNING]
> Despite its name, this key holds the user's **security token** — the dashboard calls it **Security Token** — not an OAuth access token. Salesforce issues it to the user, and RudderStack appends it to `password` to sign in.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- A spec that omits this key fails validation with `'initial_access_token' is required`.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `sandbox` — boolean, default `false`

Sign in to a Salesforce sandbox through `test.salesforce.com` instead of a production org through `login.salesforce.com`.

### Record mapping

Without an external ID naming a Salesforce object, RudderStack looks the lead up by the event's email and updates it, or creates a lead when none matches. An `identify` event without an email then fails.

#### `map_properties` — boolean, default `true`

Map traits onto standard Lead and Contact fields, such as `firstName` to `FirstName`, and send every other trait to a custom field named after it with a `__c` suffix. Other Salesforce objects receive traits as they are.

> [!WARNING]
> RudderStack doesn't read this key for this destination today, and always maps. Setting `map_properties` to `false` has no effect.

#### `use_contact_id` — boolean, default `false`

When the lead matching the event's email has been converted, update the contact it was converted to instead of the lead. The dashboard suggests turning it on when your lead and contact field mappings are the same.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Salesforce in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Salesforce accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — events reach Salesforce from RudderStack's servers, never in device mode.

The dashboard also accepts `warehouse` for Salesforce, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'salesforce-prod' (type 'salesforce') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'salesforce-prod' config has no 'connection_mode' entry for source type 'web'
```

Salesforce needs no additional config keys to connect a source of any type.

## Secrets

`password` and `initial_access_token` are the secret keys. Write each as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  password: "{{ .SALESFORCE_PASSWORD }}"
  initial_access_token: "{{ .SALESFORCE_SECURITY_TOKEN }}"
```

`user_name` isn't a secret — it doesn't sign in without the password and security token.
