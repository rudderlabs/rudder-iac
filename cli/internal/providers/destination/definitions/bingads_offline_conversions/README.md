# BingAds Offline Conversions (`bingads_offline_conversions`)

BingAds Offline Conversions is a Reverse ETL destination. RudderStack syncs rows from a warehouse to Microsoft Advertising as offline conversions — adding, restating, and retracting them as the rows change — authenticating with a Microsoft Advertising account connected through OAuth.

In a BingAds Offline Conversions destination spec:

- `type: bingads_offline_conversions`
- `definition_version: 1`

> [!NOTE]
> `bingads_offline_conversions` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: bing-offline-conversions-prod
spec:
  id: bing-offline-conversions-prod
  display_name: BingAds Offline Conversions Production
  type: bingads_offline_conversions
  definition_version: 1
  enabled: true
  config:
    rudder_account_id: 2fS9kQwPzX7rTnV4yLbH8mJcD1e
    customer_account_id: "532104745"
    customer_id: "343100598"
    is_hash_required: true

    connection_mode:
      warehouse: cloud
```

The above example has RudderStack hash email addresses and phone numbers before upload — see [`is_hash_required`](#is_hash_required--boolean-default-false). BingAds Offline Conversions takes only Reverse ETL sources, so `connection_mode` has a single `warehouse` entry, and the connection itself has to meet extra conditions — see [Connect a source](#connect-a-source).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> BingAds Offline Conversions accepts only `record` events — the rows a Reverse ETL sync produces — and runs only in `cloud` mode.

### Connection

#### `rudder_account_id` — string, required

ID of the RudderStack account that holds the OAuth connection to Microsoft Advertising. RudderStack authenticates every upload with that account, so the destination config carries no Microsoft credentials of its own.

> [!NOTE]
> Rudder CLI can't create this account. It's an OAuth connection, made by signing in to Microsoft from the RudderStack dashboard: create it there, then write its ID here. `rudder-cli workspace accounts list` lists the workspace's accounts with their IDs.

- Rudder CLI doesn't check the value — neither that the account exists nor that it's a Microsoft Advertising account.

#### `customer_account_id` — string, required

Microsoft Advertising account ID — the `aid` value in the URL of the **Campaigns** page in the Microsoft Advertising web app.

- Digits only. Write it as a string: an unquoted number fails validation.
- A `{{ path || fallback }}` template is accepted in place of a literal.

```yaml
customer_account_id: "532104745"
```

#### `customer_id` — string, required

Microsoft Advertising customer ID — the `cid` value in the same URL.

- Digits only, written as a string, on the same terms as `customer_account_id`.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `is_hash_required` — boolean, default `false`

SHA-256 hash the `email` and `phone` fields of each row before upload. The dashboard labels this **Enable it, if you are not sending hashed data.**

Leave it `false` when your warehouse already holds hashed values.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach BingAds Offline Conversions in. BingAds Offline Conversions accepts only `warehouse: cloud`.

- The entry is required to connect a Reverse ETL source — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  warehouse: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

BingAds Offline Conversions accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `warehouse` | `cloud` |

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types). BingAds Offline Conversions accepts no other source type, even in the dashboard.

## Connect a source

Only a Reverse ETL connection can reach BingAds Offline Conversions. Reverse ETL connections sit behind the `retlConnectionSupport` experimental flag — see [Source types](../README.md#source-types). A connection is checked against these rules at `validate` time.

**The source's type must be supported.** BingAds Offline Conversions accepts only `warehouse`, the type a Reverse ETL source resolves to. Every event stream source resolves to a type it doesn't accept — a JavaScript source to `web`, and webhook and server-side SDK sources to `cloud` — and reports, for example:

```text
destination 'bing-offline-conversions-prod' (type 'bingads_offline_conversions') does not support source 'my-js-source':
source type 'web' is not among supported source types: warehouse
```

**The destination config must carry a `connection_mode` entry for `warehouse`.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'bing-offline-conversions-prod' config has no 'connection_mode' entry for source type 'warehouse'
```

BingAds Offline Conversions needs no additional config keys to connect a source.

**The connection must use object mapping and the `mirror` sync behaviour.** BingAds Offline Conversions accepts only `mirror`, which only object mapping can use. A connection runs object mapping when its `config` sets `object`; without `object` it runs the JSON mapper, and no sync behaviour is left:

```text
source definition 'postgres' and destination 'bing-offline-conversions-prod' share no sync behaviour for the json_mapper flow; the connection cannot sync
```

With `object` set, `sync_behaviour` has to be `mirror`:

```text
'sync_behaviour' must be one of [mirror] for source definition 'postgres' and destination 'bing-offline-conversions-prod' on the object_mapping flow
```

Note that:

- Object mapping takes exactly one identifier, and no `constants` or `event`.
- Rudder CLI checks only that `object` is set, not that BingAds Offline Conversions offers an object of that name.
- The source's warehouse must offer `mirror` as well. S3 and SFTP sources offer only `upsert`, so they can't sync to this destination.
- The OAuth account in `rudder_account_id` isn't checked at `validate` time.

> [!WARNING]
> These checks run locally. Applying a Reverse ETL connection to an OAuth-linked destination like this one hasn't been verified against a live workspace yet, which is why `bingads_offline_conversions` stays unverified.

## Secrets

BingAds Offline Conversions has no secret keys. Rudder CLI treats none of its keys as secret, and the dashboard masks none of them. The Microsoft Advertising credentials stay with the OAuth account that `rudder_account_id` names, so a spec for this destination needs no `{{ .VAR }}` references.
