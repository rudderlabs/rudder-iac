# Mixpanel (`mp`)

Mixpanel is a product analytics destination. RudderStack sends events to Mixpanel's APIs from its servers, or — for web sources — through Mixpanel's own SDK in device mode.

In a Mixpanel destination spec:

- `type: mp`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: mixpanel-prod
spec:
  id: mixpanel-prod
  display_name: Mixpanel Production
  type: mp
  definition_version: 1
  enabled: true
  config:
    token: "{{ .MIXPANEL_TOKEN }}"
    data_residency: us
    identity_merge_api: simplified

    use_user_defined_page_event_name: true
    user_defined_page_event_template: "Viewed {{ category }} {{ name }} page"
    use_user_defined_screen_event_name: false

    set_once_properties:
      - signup_date
    prop_increments:
      - purchase_count
    group_key_settings:
      - company_id

    strict_mode: false
    drop_traits_in_track_event: false
    union_properties:
      - interests
    append_properties:
      - viewed_plans
    use_new_mapping: true
    user_deletion_api: engage

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example connects everything in `cloud` mode, so it omits the web SDK settings, which apply only to web sources in `device` mode — see [Web device mode](#web-device-mode).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Mixpanel settings split by connection mode. The groups below say which mode each key affects. Rudder CLI accepts every key whatever the mode — a key for the other mode is stored and ignored.

### Connection

#### `token` — string, required, secret

Your Mixpanel project token, from **Project Settings** > **Access Keys** in Mixpanel.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `data_residency` — string, required

Region of the Mixpanel servers RudderStack sends data to.

- One of `us`, `eu`, or `in`.
- The dashboard defaults this field to `us`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation.

#### `identity_merge_api` — string, required

Mixpanel ID merge API used to tie user activity together across devices.

- `simplified` or `original`. If you choose `simplified`, turn on Simplified ID Merge for your project in Mixpanel first.
- The dashboard defaults this field to `original`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation.

### Event naming

These keys apply in both modes.

#### `use_user_defined_page_event_name` — boolean, default `false`

Name `page` events from `user_defined_page_event_template` instead of the default name.

#### `user_defined_page_event_template` — string, default `Viewed {{ category }} {{ name }} page`

Template for `page` event names. Each `{{ field }}` is replaced with that field from the event — `category` from `properties.category`, and `name` from the event name.

- Applies when `use_user_defined_page_event_name` is `true`. Omitting it uses the default template; an explicit empty string fails validation.
- At most 200 characters, and must not contain line breaks.
- `{{ category }}`-style fields have no leading dot, so Rudder CLI's variable substitution leaves them alone.

With the default template, a `page` event named `Home` with `properties.category: Integration` becomes `Viewed Integration Home page`.

#### `use_user_defined_screen_event_name` — boolean, default `false`

Name `screen` events from `user_defined_screen_event_template` instead of the default name.

#### `user_defined_screen_event_template` — string, default `Viewed {{ category }} {{ name }} screen`

Template for `screen` event names, on the same terms as `user_defined_page_event_template`.

- Applies when `use_user_defined_screen_event_name` is `true`. Omitting it uses the default template; an explicit empty string fails validation.
- At most 200 characters, and must not contain line breaks.

### User profiles and groups

These keys apply in both modes. Each takes an array of property or trait names, each at most 100 characters or a `{{ path || fallback }}` template.

#### `set_once_properties` — array of strings

`identify` traits Mixpanel sets on a user profile once and never overwrites.

#### `prop_increments` — array of strings

Numeric properties to increment on the user's Mixpanel People profile.

#### `group_key_settings` — array of strings

Group keys that identify groups in Mixpanel. RudderStack sends `group` calls only when at least one key is listed.

### Cloud mode

These keys apply to events sent in `cloud` mode. The dashboard shows them only when the destination has a cloud-mode connection.

#### `strict_mode` — boolean, default `false`

Have Mixpanel validate each request and return an error for every event that fails, rather than accepting what it can.

#### `drop_traits_in_track_event` — boolean, default `false`

Drop the persisted user traits (`context.traits`) from `track` events. When `false`, they're sent alongside the event properties.

#### `union_properties` — array of strings

`identify` traits whose values are added to a list property on the profile only if not already present.

#### `append_properties` — array of strings

`identify` traits whose values are appended to a list property on the profile.

#### `use_new_mapping` — boolean, default `false`

Send name traits in snake case — `$first_name` and `$last_name` — instead of the older `$firstName` and `$lastName`. The old mapping is being deprecated, so turn this on.

#### `user_deletion_api` — string, default `engage`

Mixpanel API RudderStack uses to delete a user.

- `engage` — deletes the user profile but keeps its events.
- `task` — deletes the profile and its events. Needs `gdpr_api_token`.

#### `gdpr_api_token` — string, required, secret

Mixpanel GDPR API token, used to delete a user's profile and events.

- Required when `user_deletion_api` is `task`. Leave it unset otherwise.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `service_account_user_name` — string, secret, internal

Username of a Mixpanel service account.

- Not validated locally.

This key, `service_account_secret`, and `project_id` are a Mixpanel service account credential, used together. `token` is required either way, so set these only if your destination already uses them.

#### `service_account_secret` — string, secret, internal

Secret of the Mixpanel service account named in `service_account_user_name`.

- Not validated locally.

#### `project_id` — string, internal

Mixpanel project ID, sent with service account authentication.

- Not validated locally.

### Web device mode

These keys configure Mixpanel's SDK in the browser, so they apply only to web sources connected in `device` mode.

#### `people` — boolean, default `false`

Send `identify` calls to Mixpanel People. This is what the dashboard calls **Use Mixpanel People**.

#### `set_all_traits_by_default` — boolean, default `false`

Set every `identify` trait as a super property, and — when `people` is `true` — as a People property too.

#### `super_properties` — array of strings

Event properties to set as Mixpanel super properties.

#### `people_properties` — array of strings

`identify` traits to set as Mixpanel People properties.

#### `event_increments` — array of strings

Event names whose occurrences are counted on the user's People profile.

#### `consolidated_page_calls` — boolean, default `true`

Send every `page` call as a `Loaded a Page` event. Mixpanel recommends leaving this on.

#### `track_categorized_pages` — boolean, default `false`

Also send an event for each `page` call that has a category — `page("Docs", "Index")` becomes `Viewed Docs Index Page`.

#### `track_named_pages` — boolean, default `false`

Also send an event for each `page` call that has a name — `page("Signup")` becomes `Viewed Signup Page`.

#### `source_name` — string

Value sent as `rudderstack_source_name` with every event, `page`, and `screen` call.

- At most 100 characters, or a `{{ path || fallback }}` template.

#### `session_replay_percentage` — object

Percentage of SDK initializations that qualify for session replay, sent to Mixpanel as `record_sessions_percent`. When unset, no sessions are recorded.

- Keyed by `web`, the only source type it applies to.
- The value is a whole number from `0` to `100`, written as a string — `"20"`, not `20`. A template is accepted.

```yaml
session_replay_percentage:
  web: "20"
```

#### `ignore_dnt` — boolean, default `false`

Have Mixpanel's SDK ignore the browser's **Do Not Track** setting.

#### `cross_subdomain_cookie` — boolean, default `false`

Persist the Mixpanel cookie across subdomains of your site.

#### `persistence_type` — string, default `cookie`

Where Mixpanel's SDK persists its state.

- `none`, `cookie`, or `localStorage`. With `localStorage`, an existing Mixpanel cookie of the same name is moved into local storage.

#### `persistence_name` — string

Suffix added to the Mixpanel cookie name.

- At most 100 characters, or a `{{ path || fallback }}` template.

#### `secure_cookie` — boolean, default `false`

Mark the Mixpanel cookie as secure, so it's sent only over HTTPS.

### Event filtering

> [!WARNING]
> Client-side event filtering is applied by the RudderStack SDK, so it affects only web sources connected in `device` mode. Events sent in `cloud` mode aren't filtered by it.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to Mixpanel, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Mixpanel in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).
- A mode the source type doesn't support on this destination fails validation — for example `device` for `android`.

```yaml
connection_mode:
  web: device
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Mixpanel accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud` |
| `ios_swift` | `cloud` |
| `web` | `cloud`, `device` |
| `unity` | `cloud` |
| `cloud` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |
| `warehouse` | `cloud` |

Only `web` offers `device` mode, which loads Mixpanel's SDK in the browser. In `cloud` mode Mixpanel accepts `alias`, `group`, `identify`, `page`, `screen`, and `track`; in `device` mode, web sources send `alias`, `group`, `identify`, `page`, and `track`.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every type an event stream or Reverse ETL source can resolve to is supported here, so this check always passes.

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'mixpanel-prod' config has no 'connection_mode' entry for source type 'web'
```

Mixpanel needs no additional config keys to connect a source of any type, in any mode.

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it.

## Secrets

Rudder CLI treats four keys as secrets: `token`, `gdpr_api_token`, `service_account_user_name`, and `service_account_secret`. Write each one you use as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  token: "{{ .MIXPANEL_TOKEN }}"
  gdpr_api_token: "{{ .MIXPANEL_GDPR_TOKEN }}"
```

Note that:

- Like every secret key, these show as changed on each plan and are re-sent on apply, because Rudder CLI never compares secrets with the remote. That's expected, not drift.
- In device mode the project token is embedded in the page's JavaScript, so masking it protects your YAML, not the value itself.
