# LinkedIn Insight Tag (`linkedin_insight_tag`)

LinkedIn Insight Tag is a web device mode advertising destination. RudderStack's JavaScript SDK loads the Insight Tag in the browser, where it tracks page loads on its own, and fires a LinkedIn conversion for each `track` event you map to a conversion ID.

In a LinkedIn Insight Tag destination spec:

- `type: linkedin_insight_tag`
- `definition_version: 1`

> [!NOTE]
> `linkedin_insight_tag` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: linkedin-insight-tag-prod
spec:
  id: linkedin-insight-tag-prod
  display_name: LinkedIn Insight Tag Production
  type: linkedin_insight_tag
  definition_version: 1
  enabled: true
  config:
    partner_id: "1234567"
    event_to_conversion_id_map:
      - from: Order Completed
        to: "12345678"
      - from: Signed Up
        to: "23456789"

    connection_mode:
      web: device
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
```

The above example sends LinkedIn a conversion for `Order Completed` and `Signed Up` events, and nothing for any other `track` event — see [Conversion tracking](#conversion-tracking). The partner ID and conversion IDs are quoted, because each is a string of digits.

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> LinkedIn Insight Tag runs only in web device mode, and acts only on `track` calls from the SDK. Every key on this page configures what the Insight Tag does in the browser.

### Connection

#### `partner_id` — string, required

Your LinkedIn partner ID — the `_linkedin_partner_id` value in LinkedIn's Insight Tag code. Each Insight Tag has its own partner ID; if it doesn't match the one in your ad account, the tag collects neither page loads nor conversions.

- Write it in quotes: `"1234567"`, not `1234567`. An unquoted number fails validation.
- Not validated locally beyond being present.
- The dashboard masks this field, but Rudder CLI doesn't treat it as a secret — see [Secrets](#secrets).

### Conversion tracking

#### `event_to_conversion_id_map` — array of objects

Maps `track` event names to LinkedIn conversion IDs. When a mapped event fires, the SDK sends LinkedIn one conversion for each ID mapped to that event. `track` events with no mapping aren't sent to LinkedIn at all.

- `from` — RudderStack event name. Matched case-sensitively, ignoring leading and trailing spaces: `Order Completed` and `order completed` are different events.
- `to` — ID of an event-specific conversion, from the conversion's setup in LinkedIn Campaign Manager. Write it in quotes — an unquoted number fails validation.
- Map one `from` to several `to` values to fire several conversions for one event.
- Each field is at most 100 characters, or a `{{ path || fallback }}` template.
- Rudder CLI doesn't require either field, but an entry fires a conversion only with both.

```yaml
event_to_conversion_id_map:
  - from: Order Completed
    to: "12345678"
  - from: Order Completed
    to: "87654321"
```

### Event filtering

#### `event_filtering` — object

Restricts which `track` events the SDK passes to the Insight Tag, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.
- Omit the block entirely to filter nothing.

Filtering runs before the conversion mapping. An event it drops fires no conversion, even when mapped, and an event it allows fires one only when mapped — so a `blacklist` entry pauses a mapped conversion without removing its mapping.

```yaml
event_filtering:
  blacklist:
    - Signed Up
```

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach LinkedIn in. LinkedIn Insight Tag accepts only `web: device`.

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: device
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

LinkedIn Insight Tag accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `web` | `device` |

LinkedIn Insight Tag accepts only web sources, and only in `device` mode — the SDK loads the Insight Tag in the browser, and no events pass through RudderStack's servers.

LinkedIn Insight Tag doesn't accept `warehouse`, even in the dashboard.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** LinkedIn Insight Tag accepts only `web`, the type a JavaScript source resolves to. Every other source resolves to a type it doesn't accept — an iOS source to `ios`, an Android source to `android`, and webhook and server-side SDK sources to `cloud` — and reports, for example:

```text
destination 'linkedin-insight-tag-prod' (type 'linkedin_insight_tag') does not support source 'my-ios-app': source type 'ios' is not among supported source types: web
```

A Reverse ETL connection, whose source resolves to `warehouse`, is refused the same way:

```text
destination 'linkedin-insight-tag-prod' (type 'linkedin_insight_tag') does not accept rETL sources: source type 'warehouse' is not among supported source types: web
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'linkedin-insight-tag-prod' config has no 'connection_mode' entry for source type 'web'
```

LinkedIn Insight Tag needs no additional config keys to connect a web source.

## Secrets

Rudder CLI treats none of LinkedIn Insight Tag's keys as secret.

The dashboard masks `partner_id` (**Partner ID**). Rudder CLI doesn't treat it as a secret, so it's written to YAML in plain text. The partner ID is embedded in the page's JavaScript in any case, so masking it would protect your YAML, not the value itself.
