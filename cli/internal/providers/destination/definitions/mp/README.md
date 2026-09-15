# Mixpanel (`mp`)

Streaming destination. Mixpanel receives events either from RudderStack's
servers (cloud mode) or directly from the Mixpanel SDK loaded on your site
(device mode, web only).

In a destination spec:

- `type: mp`
- `definition_version: 1`

## Modes

The mode a source uses is set per source type in `config.connection_mode`. Only
`web` accepts `device`; every other source type is cloud only — see
[Source types](#source-types).

Each key below is badged with where it applies:

- `cloud` — events sent from RudderStack's servers
- `device` — events sent by the Mixpanel SDK; supported platforms follow the
  badge where support is limited to some of them

A key is accepted by `validate` whatever your sources use; the badges tell you
where the setting takes effect.

## Example

```yaml
version: rudder/v1
kind: destination
metadata:
  name: mixpanel
spec:
  id: mixpanel
  display_name: Mixpanel
  type: mp
  definition_version: 1
  enabled: true
  config:
    token: "{{ .MIXPANEL_TOKEN }}"
    data_residency: us
    identity_merge_api: simplified
    project_id: "2100000"
    service_account_user_name: rudder-service-account
    service_account_secret: "{{ .MIXPANEL_SERVICE_ACCOUNT_SECRET }}"

    user_deletion_api: task
    gdpr_api_token: "{{ .MIXPANEL_GDPR_API_TOKEN }}"

    use_user_defined_page_event_name: true
    user_defined_page_event_template: "Viewed a Page"
    use_user_defined_screen_event_name: false
    user_defined_screen_event_template: "Viewed a Screen"

    people: true
    set_once_properties:
      - signup_date
    union_properties:
      - viewed_categories
    append_properties:
      - recent_searches
    prop_increments:
      - login_count
    group_key_settings:
      - company_id
    drop_traits_in_track_event: false

    strict_mode: true
    use_new_mapping: false
    event_filtering:
      whitelist:
        - Order Completed
        - Product Viewed

    # Device mode (web)
    consolidated_page_calls: true
    track_categorized_pages: false
    track_named_pages: false
    set_all_traits_by_default: false
    super_properties:
      - plan
    people_properties:
      - email
    event_increments:
      - Order Completed
    persistence_type: localStorage
    persistence_name: rudder_mp
    cross_subdomain_cookie: false
    secure_cookie: true
    ignore_dnt: false
    source_name: my-website
    session_replay_percentage:
      web: "10"

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

#### `token` — string, required, secret
`cloud` `device` · web

Mixpanel API token. At most 100 characters.

Supply it as a `{{ .VAR }}` reference — see [Secrets](#secrets).

#### `data_residency` — string, required
`cloud` `device` · web

Mixpanel server region. One of `us`, `eu` or `in`.

#### `identity_merge_api` — string, required
`cloud` `device` · web

Mixpanel identity merge type. One of `simplified` or `original`.

#### `project_id` \* — string
`cloud`

Mixpanel project ID, used by the server-side user deletion API.

#### `service_account_user_name` \* — string
`cloud`

Username of the Mixpanel service account used for server-side API calls.

#### `service_account_secret` \* — string, secret
`cloud`

Secret of the Mixpanel service account. Supply it as a `{{ .VAR }}` reference.

### User deletion

#### `user_deletion_api` \* — string, default `engage`
`cloud`

Which Mixpanel API handles user deletion requests. One of `engage` or `task`.

#### `gdpr_api_token` \* — string, secret
`cloud`

Token for Mixpanel's GDPR deletion API. **Required when `user_deletion_api` is
`task`.** At most 100 characters. Supply it as a `{{ .VAR }}` reference.

### Page and screen tracking

#### `use_user_defined_page_event_name` — boolean, default `false`
`cloud` `device` · web

Use a user-defined event name for `page` calls.

#### `user_defined_page_event_template` — string, default `Viewed {{ category }} {{ name }} page`
`cloud` `device` · web

Template for the user-defined page event name. **Required when
`use_user_defined_page_event_name` is `true`.** At most 200 characters.

#### `use_user_defined_screen_event_name` — boolean, default `false`
`cloud`

Use a user-defined event name for `screen` calls.

#### `user_defined_screen_event_template` — string, default `Viewed {{ category }} {{ name }} screen`
`cloud`

Template for the user-defined screen event name. **Required when
`use_user_defined_screen_event_name` is `true`.** At most 200 characters.

### Traits and properties

Each list below takes trait or property names, at most 100 characters per entry.

#### `people` — boolean, default `false`
`cloud` `device` · web

Send all `identify` calls to Mixpanel's People feature.

#### `set_once_properties` — string array
`cloud` `device` · web

Properties to set only once.

#### `union_properties` — string array
`cloud`

Properties to union.

#### `append_properties` — string array
`cloud`

Properties to append.

#### `prop_increments` — string array
`cloud` `device` · web

Properties to increment in People.

#### `group_key_settings` — string array
`cloud` `device` · web

Group keys.

#### `drop_traits_in_track_event` — boolean, default `false`
`cloud`

Drop traits from `track` event calls.

### Other

#### `strict_mode` — boolean, default `false`
`cloud`

Enable Mixpanel's strict mode, which rejects malformed events rather than
accepting them silently.

#### `use_new_mapping` — boolean, default `false`
`cloud`

Map camelCase fields to snake_case when sending to Mixpanel.

#### `event_filtering` — object
`cloud` `device`

Determine which events are blocked or allowed to flow through to Mixpanel.
Exactly one of the two lists may be set — declaring both fails validation.

- `whitelist` — event names to allowlist
- `blacklist` — event names to denylist

Each entry is at most 100 characters. Omit the block to send every event.

## Device mode only

The keys below configure the Mixpanel SDK and apply only to a web source in
device mode.

`validate` accepts these keys whatever sources the project connects, so you can
declare them ahead of connecting a web source.

### Page tracking

#### `consolidated_page_calls` — boolean, default `true`
`device` · web

Track a `Loaded a Page` event for every `page` call. On by default, matching
Mixpanel's own recommendation.

#### `track_categorized_pages` — boolean, default `false`
`device` · web

Track events for `page` calls that carry a category — `page('Docs', 'Index')`
becomes `Viewed Docs Index Page`.

#### `track_named_pages` — boolean, default `false`
`device` · web

Track events for `page` calls that carry a name — `page('Signup')` becomes
`Viewed Signup Page`.

### Traits and properties

#### `set_all_traits_by_default` — boolean, default `false`
`device` · web

Set all traits on `identify` calls as super properties, and as people properties
when `people` is also enabled.

#### `super_properties` — string array
`device` · web

Properties to send as super properties. Each entry at most 100 characters.

#### `people_properties` — string array
`device` · web

Traits to set as People properties. Each entry at most 100 characters.

#### `event_increments` — string array
`device` · web

Events to increment in People. Each entry at most 100 characters.

### Cookies and persistence

#### `persistence_type` — string, default `cookie`
`device` · web

Where the Mixpanel SDK persists its state. One of `none`, `cookie` or
`localStorage`.

#### `persistence_name` — string
`device` · web

Name of the Mixpanel persistence store. At most 100 characters.

#### `cross_subdomain_cookie` — boolean, default `false`
`device` · web

Allow the Mixpanel cookie to persist across subdomains of your application.

#### `secure_cookie` — boolean, default `false`
`device` · web

Mark the Mixpanel cookie as secure, so it is only transmitted over HTTPS.

### Other

#### `ignore_dnt` — boolean, default `false`
`device` · web

Ignore the browser's "Do Not Track" setting.

#### `source_name` — string
`device` · web

When set, sent as `rudderstack_source_name` on every event, page and screen
call. At most 100 characters.

#### `session_replay_percentage` — object
`device` · web

Percentage of SDK initialisations that qualify for session replay capture.
Object with a single `web` key, written as a digit string.

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

#### `connection_mode` \* — object

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
destination 'mixpanel' (type 'mp') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'mixpanel' config has no 'connection_mode' entry for source type 'web'
```

Mixpanel requires no additional config keys to connect a source of any type.

## Secrets

`token`, `service_account_secret` and `gdpr_api_token` are secret keys. Write
each as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
token: "{{ .MIXPANEL_TOKEN }}"
service_account_secret: "{{ .MIXPANEL_SERVICE_ACCOUNT_SECRET }}"
gdpr_api_token: "{{ .MIXPANEL_GDPR_API_TOKEN }}"
```

```sh
export RUDDER_MIXPANEL_TOKEN=...
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`rudder-cli import` writes each back as a `{{ .VAR }}` placeholder rather than
its value, since the API does not return secrets. Fill the placeholders in
before the first apply.
