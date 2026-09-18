# PostHog (`posthog`)

Streaming destination. PostHog receives events either from RudderStack's servers
(cloud mode) or directly from the PostHog SDK loaded on your site (device mode,
web only).

In a destination spec:

- `type: posthog`
- `definition_version: 1`

## Modes

The mode a source uses is set per source type in `config.connection_mode`. Only
`web` accepts `device`; every other source type is cloud only — see
[Source types](#source-types).

Each key below is badged with where it applies:

- `cloud` — events sent from RudderStack's servers
- `device` — events sent by the PostHog SDK

A key is accepted by `validate` whatever your sources use; the badges tell you
where the setting takes effect.

## Example

```yaml
version: rudder/v1
kind: destination
metadata:
  name: posthog
spec:
  id: posthog
  display_name: PostHog
  type: posthog
  definition_version: 1
  enabled: true
  config:
    api_key: "{{ .POSTHOG_API_KEY }}"
    endpoint: "https://eu.posthog.com"
    use_v2_group: false

    event_filtering:
      whitelist:
        - Order Completed
        - Product Viewed

    # Device mode (web)
    autocapture:
      web: true
    capture_page_view:
      web: true
    disable_session_recording:
      web: false
    enable_local_storage_persistence:
      web: true
    person_profiles:
      web: identified_only
    xhr_headers:
      - key: X-Tenant
        value: acme
    property_blacklist:
      - property: email
      - property: phone

    connection_mode:
      web: device
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

## Config keys

`config` accepts only the keys documented here — anything else fails validation
with `unknown config field "<key>"`.

Keys that declare a default are filled in before the spec enters the resource
graph, matching what the backend stores, so omitting one is equivalent to
writing its default and does not produce a permanent diff.

A `*` after a key name marks a description written without a Terraform provider
source to draw on. Those need a closer review pass; the markers come out once the
wording is confirmed.

### Connection

#### `api_key` — string, required, secret
`cloud` `device` · web

Your PostHog API key. At most 100 characters.

Unlike most string keys here, this one does not accept a
`{{ path || fallback }}` template. Supply it as a `{{ .VAR }}` reference, which
is substituted before validation runs — see [Secrets](#secrets).

#### `endpoint` — string
`cloud` `device` · web

The endpoint of your PostHog service. At most 100 characters, and must not
contain `.ngrok.io`.

### Event delivery

#### `use_v2_group` — boolean, default `false`
`cloud`

Use v2 grouping for your PostHog service.

#### `event_filtering` — object
`cloud` `device`

Filter which events are sent to PostHog. Exactly one of the two lists may be set
— declaring both fails validation.

- `whitelist` — event names to allowlist
- `blacklist` — event names to denylist

Each entry is at most 100 characters. Omit the block to send every event.

## Device mode only

The keys below configure the PostHog Web SDK and apply only to a web source in
device mode.

`validate` accepts these keys whatever sources the project connects, so you can
declare them ahead of connecting a web source.

#### `autocapture` — object
`device` · web

Let PostHog automatically capture interactions on the page.

#### `capture_page_view` — object
`device` · web

Capture page view events.

#### `disable_session_recording` — object
`device` · web

Turn off PostHog session recording.

#### `enable_local_storage_persistence` — object
`device` · web

Persist PostHog state in local storage rather than cookies.

#### `person_profiles` — object, default `{web: always}`
`device` · web

When PostHog creates person profiles. `always` creates one for every user;
`identified_only` creates them only for identified users.

#### `xhr_headers` \* — array of objects
`device` · web

Extra headers the SDK adds to the requests it sends to PostHog. Each entry takes
a `key` and a `value`, each at most 100 characters.

#### `property_blacklist` — array of objects
`device` · web

Properties the SDK strips from events before sending them. Each entry takes a
`property` name.

## Source types

| Source type | Modes |
| --- | --- |
| `web` | cloud, device |
| `android` | cloud |
| `android_kotlin` | cloud |
| `ios` | cloud |
| `ios_swift` | cloud |
| `unity` | cloud |
| `react_native` | cloud |
| `flutter` | cloud |
| `cordova` | cloud |
| `cloud` | cloud |

## Per-source keys

Both keys below are objects keyed by the local source type. A key naming a
source type this destination does not support fails validation.

#### `connection_mode` — object

Selects the mode per source type. Values are constrained to the modes that
source type supports, so `device` is rejected for every type except `web`:

```yaml
connection_mode:
  web: device
  cloud: cloud
```

#### `consent_management` — object

Specify consent configuration data for multiple providers, per source type. The
entry shape, accepted providers, and the rules on `resolution_strategy` and
`consents` are shared across all destinations and documented in
[../common/README.md](../common/README.md).

## Connecting a source

An event stream connection to this destination is checked against two rules at
`validate` time.

**The source's type must be supported.** A source's type is mapped to one of the
tokens above first — a JavaScript source resolves to `web`, and webhook and
server-side SDK sources resolve to `cloud`. An unsupported type reports:

```
destination 'posthog' (type 'posthog') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'posthog' config has no 'connection_mode' entry for source type 'web'
```

PostHog requires no additional config keys to connect a source of any type.

## Secrets

`api_key` is the only secret key. Write it as a `{{ .VAR }}` reference and supply
the value at apply time:

```yaml
api_key: "{{ .POSTHOG_API_KEY }}"
```

```sh
export RUDDER_POSTHOG_API_KEY=...
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`rudder-cli import` writes `api_key` back as a `{{ .VAR }}` placeholder rather
than its value, since the API does not return secrets. Fill the placeholder in
before the first apply.
