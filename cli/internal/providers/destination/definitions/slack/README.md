# Slack (`slack`)

Slack is a business messaging destination. RudderStack turns `identify` and `track` events into messages and posts them to Slack channels through incoming webhooks, from its servers.

In a Slack destination spec:

- `type: slack`
- `definition_version: 1`

> [!NOTE]
> `slack` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: slack-prod
spec:
  id: slack-prod
  display_name: Slack Production
  type: slack
  definition_version: 1
  enabled: true
  config:
    webhook_url: "{{ .SLACK_WEBHOOK_URL }}"

    incoming_webhooks_type: modern
    event_channel_settings:
      - name: Order Completed
        webhook: "{{ .SLACK_ORDERS_WEBHOOK_URL }}"
        regex: false
      - name: "^Trial (Started|Ended)$"
        webhook: "{{ .SLACK_GROWTH_WEBHOOK_URL }}"
        regex: true

    identify_template: "Identified {{name}}{{newline}}Plan: {{plan}}"
    event_template_settings:
      - name: Order Completed
        template: "{{name}} completed an order worth {{revenue}}"
        regex: false

    whitelisted_trait_settings:
      - email
      - plan
    deny_list_of_events:
      - Heartbeat

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example routes `track` events by webhook URL, so each channel entry carries its own `webhook` — see [Channel routing](#channel-routing). The webhook URLs are `{{ .VAR }}` references even though Rudder CLI doesn't treat them as secrets — see [Secrets](#secrets).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Slack accepts only `identify` and `track` events. RudderStack doesn't deliver any other event type to it.

### Connection

#### `webhook_url` — string, required

Slack incoming webhook URL. RudderStack posts every `identify` message here, and every `track` message that [channel routing](#channel-routing) doesn't send elsewhere.

- At most 100 characters, and must not contain line breaks.
- Any value containing `.ngrok.io` is rejected.
- A spec that omits this key fails validation with `'webhook_url' is required`.

### Channel routing

`event_channel_settings` sends `track` events to channels other than the one `webhook_url` posts to, and `incoming_webhooks_type` decides how. `identify` messages always go to `webhook_url`.

#### `incoming_webhooks_type` — string, default `legacy`

How a matching `event_channel_settings` entry reaches its channel:

- `legacy` — RudderStack posts through `webhook_url` and names the entry's `channel` in the message. The dashboard marks this method as due for deprecation.
- `modern` — RudderStack posts to the entry's own `webhook`, a URL from a Slack app with incoming webhooks enabled for that channel.

#### `event_channel_settings` — array of objects

Routes `track` events to channels by event name. RudderStack uses the first entry that matches, and posts an event that matches none through `webhook_url`.

- `name` — event name to match. At most 100 characters, and must not contain line breaks. It also must not contain a parenthesized group that holds `*`, `+`, `{`, or `|` and is followed by `*`, `+`, `?`, or `{` — such as `(a|b)+` — even when `regex` is `false`.
- `regex` — boolean. `true` treats `name` as a regular expression, which matches anywhere in the event name — anchor it with `^` and `$` to match the whole name. Omit it, or set `false`, for an exact match.
- `channel` — channel to post to when `incoming_webhooks_type` is `legacy`: a channel name such as `#orders`, or `@user` for a direct message. At most 100 characters, and must not contain line breaks. Quote it, since an unquoted `#` starts a YAML comment.
- `webhook` — incoming webhook URL to post to when `incoming_webhooks_type` is `modern`. Same limits as `webhook_url`, including the `.ngrok.io` ban.

Rudder CLI doesn't check that an entry carries the field its webhook type uses. RudderStack skips an entry without it, as though its `name` never matched.

```yaml
incoming_webhooks_type: legacy
event_channel_settings:
  - name: Order Completed
    channel: "#orders"
  - name: "^Trial (Started|Ended)$"
    channel: "#growth"
    regex: true
```

### Message templates

Both template keys take Handlebars expressions. Write each template in quotes: an unquoted value starting with `{` isn't read as a string in YAML. Rudder CLI's `{{ .VAR }}` substitution claims only tokens with a leading dot, so `{{name}}` reaches Slack as written.

A template can use:

- `{{name}}` — the user's name, falling back to their username or email, then to `User <userId>`, or `Anonymous user <anonymousId>`.
- `{{event}}` — the event name. `track` only.
- `{{<property>}}` or `{{properties.<property>}}` — an event property, such as `{{revenue}}`. `track` only.
- `{{<trait>}}` — a trait, such as `{{plan}}`. `identify` only; `{{traitsList.<trait>}}` works in both.
- `{{traits}}` — the user's traits as `key: value` text, limited to `whitelisted_trait_settings` when it's set.
- `{{propertiesList}}` — every event property as `key: value` text. `track` only.
- `{{newline}}` — a line break. Templates can't contain literal line breaks, so this is the only way to write a multi-line message.

#### `identify_template` — string

Template for the message an `identify` event posts. When omitted, RudderStack posts `Identified {{name}}` followed by the user's traits as `key: value` pairs, limited to `whitelisted_trait_settings` when it's set.

- At most 1000 characters, and must not contain line breaks.

```yaml
identify_template: "Identified {{name}}{{newline}}Plan: {{plan}}"
```

#### `event_template_settings` — array of objects

Templates for the messages `track` events post, by event name. RudderStack uses the first entry that matches. An event that matches none posts `{{name}} did {{event}}`.

- `name`, `regex` — match events on the same terms as [`event_channel_settings`](#event_channel_settings--array-of-objects).
- `template` — the message template. At most 1000 characters, and must not contain line breaks.

```yaml
event_template_settings:
  - name: Order Completed
    template: "{{name}} completed an order worth {{revenue}}"
```

### Filtering

#### `whitelisted_trait_settings` — array of strings

Traits RudderStack lists in the default `identify` message and in `{{traits}}`. When empty, every trait is listed. A template can still name any trait directly.

- Each trait name is at most 100 characters, and must not contain line breaks.

```yaml
whitelisted_trait_settings:
  - email
  - plan
```

#### `deny_list_of_events` — array of strings

`track` event names RudderStack doesn't post to Slack. A match is exact and case-sensitive, and RudderStack fails a matching event with a configuration error instead of posting it.

- Each name is at most 100 characters, and must not contain line breaks.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Slack in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Slack accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — RudderStack posts to Slack from its servers, never in device mode.

The dashboard also accepts `warehouse` for Slack, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'slack-prod' (type 'slack') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'slack-prod' config has no 'connection_mode' entry for source type 'web'
```

Slack needs no additional config keys to connect a source of any type.

## Secrets

Slack has no secret keys. Rudder CLI writes `webhook_url` and each entry's `webhook` to YAML in plain text, and `import` writes them as literals.

An incoming webhook URL lets anyone who holds it post to its channel, so keep it out of version control with a `{{ .VAR }}` reference anyway, as the sample does — [Secrets](../README.md#secrets) covers supplying the value:

```yaml
config:
  webhook_url: "{{ .SLACK_WEBHOOK_URL }}"
```

Replace the literal URLs in an imported spec with references before you commit it.
