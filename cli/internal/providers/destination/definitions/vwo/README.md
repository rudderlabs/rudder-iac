# VWO (`vwo`)

VWO (Visual Website Optimizer) is an A/B testing destination that runs only in web device mode. RudderStack's JavaScript SDK passes `identify` traits and `track` events to VWO in the browser, prefixing each trait and event name with `rudder.`, and can report the variations VWO applies back to RudderStack as events.

In a VWO destination spec:

- `type: vwo`
- `definition_version: 1`

> [!NOTE]
> `vwo` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: vwo-prod
spec:
  id: vwo-prod
  display_name: VWO Production
  type: vwo
  definition_version: 1
  enabled: true
  config:
    account_id: "654321"

    send_experiment_track: true
    send_experiment_identify: true

    is_spa: true
    library_tolerance: "2500"
    settings_tolerance: "2000"
    use_existing_jquery: false

    event_filtering:
      whitelist:
        - Order Completed
        - Signed Up

    connection_mode:
      web: device
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example sets the SmartCode keys, which take effect only when the page has the SDK load VWO's SmartCode — see [SmartCode settings](#smartcode-settings).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> VWO runs only in web device mode. The SDK forwards `identify` and `track` calls to VWO, and also records an `Order Completed` event's `total` — or `revenue`, when `total` is absent — as a VWO revenue conversion.

### Connection

#### `account_id` — string, required

Your VWO account ID, from **Account Details** in VWO's account settings. The SDK uses it when it loads VWO's SmartCode itself — see [SmartCode settings](#smartcode-settings).

- Written as a string. Quote a numeric ID: `"654321"`, not `654321`.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- A spec that omits this key fails validation with `'account_id' is required`.

```yaml
account_id: "654321"
```

### Experiment reporting

These keys have the SDK report each variation VWO applies to a visitor back to RudderStack, so the other destinations connected to the source can analyze experiment results.

> [!WARNING]
> `send_experiment_identify` takes effect only when `send_experiment_track` is also `true` — the SDK listens for applied variations only when `send_experiment_track` is on. Set both to get the identify traits:
>
> ```yaml
> send_experiment_track: true
> send_experiment_identify: true
> ```

#### `send_experiment_track` — boolean, default `false`

Send an `Experiment Viewed` `track` event whenever VWO applies a variation, with the experiment ID, campaign name, variation ID, and variation name as properties. The dashboard calls it **Send experiment viewed as track**.

#### `send_experiment_identify` — boolean, default `false`

Send an `identify` call whenever VWO applies a variation, with a trait named `Experiment: <experiment ID>` holding the variation name. The dashboard calls it **Send experiment viewed as identify traits**.

### SmartCode settings

These keys configure VWO's SmartCode snippet. The SDK adds SmartCode to the page — with `account_id` and these settings — only when the page's `load` call opts in through its integration options:

```javascript
rudderanalytics.load(WRITE_KEY, DATA_PLANE_URL, {
  integrations: {
    VWO: {
      loadIntegration: true
    }
  }
});
```

Without that option, the SDK expects VWO's SmartCode to be on the page already, and neither `account_id` nor the four keys below have any effect.

#### `is_spa` — boolean, default `false`

Tell SmartCode the page is a single-page application. The dashboard calls it **Single Page Application (SPA)?**.

#### `library_tolerance` — string

Maximum time, in milliseconds, SmartCode waits for the VWO library to load before it shows the original page. The dashboard calls it **Library Tolerance**.

- Digits, written as a string: `"2500"`, not `2500`. Not validated as a number locally.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- The dashboard defaults it to `"2500"`; Rudder CLI doesn't fill it in.

```yaml
library_tolerance: "2500"
```

#### `settings_tolerance` — string

Maximum time, in milliseconds, SmartCode waits for campaign settings before it shows the original page. The dashboard calls it **Setting Tolerance**.

- Digits, written as a string: `"2000"`, not `2000`. Not validated as a number locally.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- The dashboard defaults it to `"2000"`; Rudder CLI doesn't fill it in. Without it, SmartCode doesn't wait for settings before showing the page.

#### `use_existing_jquery` — boolean, default `false`

Use the jQuery already on the page instead of having VWO load its own. VWO needs jQuery on the page either way. The dashboard calls it **Use Existing jquery**.

### Event filtering

Every source connects to VWO in `device` mode, where the SDK applies the filter, so the filter applies to `track` events from every source.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to VWO, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach VWO in. VWO accepts only `web: device`.

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: device
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

VWO accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `web` | `device` |

VWO accepts only web sources, and only in `device` mode — the SDK runs VWO in the browser, and no events pass through RudderStack's servers.

VWO doesn't accept `warehouse`, even in the dashboard.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** VWO accepts only `web`, the type a JavaScript source resolves to. Every other source resolves to a type it doesn't accept — an iOS source to `ios`, an Android source to `android`, and webhook and server-side SDK sources to `cloud` — and reports, for example:

```text
destination 'vwo-prod' (type 'vwo') does not support source 'my-ios-app':
source type 'ios' is not among supported source types: web
```

A Reverse ETL connection, whose source resolves to `warehouse`, is refused too:

```text
destination 'vwo-prod' (type 'vwo') does not accept rETL sources: source type 'warehouse' is not among supported source types: web
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'vwo-prod' config has no 'connection_mode' entry for source type 'web'
```

VWO needs no additional config keys to connect a web source.

## Secrets

VWO has no secret keys. Rudder CLI treats none of its keys as secret, and the dashboard masks none of them, so `import` writes every value to YAML in plain text — including `account_id`, which the page's JavaScript carries anyway.
