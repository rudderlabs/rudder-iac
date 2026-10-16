# Redis (`redis`)

Redis is a key-value store destination. RudderStack writes the traits from each `identify` event to the user's key in Redis — `user:<userId>` — so you can read a user's latest profile in real time. Redis receives only `identify` events, and only those with a `userId`; one without fails.

In a Redis destination spec:

- `type: redis`
- `definition_version: 1`

> [!NOTE]
> `redis` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: redis-prod
spec:
  id: redis-prod
  display_name: Redis Production
  type: redis
  definition_version: 1
  enabled: true
  config:
    address: redis.internal.example.com:6379
    password: "{{ .REDIS_PASSWORD }}"
    cluster_mode: false
    database: "2"
    prefix: rudderstack

    secure: true
    skip_verify: false
    ca_certificate: "{{ .REDIS_CA_CERTIFICATE }}"

    use_json_module: false

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example connects to a single Redis server over TLS. `cluster_mode` defaults to `true`, so a single server needs `cluster_mode: false`, which also lets it pick a `database` — see [Connection](#connection).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> A `{{ path || fallback }}` template resolves per event, so only `prefix` can use one. Validation also accepts a template in `address` and `database`, but RudderStack connects with those values as written.

### Connection

#### `address` — string, required

Address of your Redis server, as `host:port`. With `cluster_mode` on, list several cluster nodes separated by commas.

- At most 100 characters, and must not contain `.ngrok.io`.

#### `password` — string, secret

Password RudderStack authenticates to Redis with. Omit it for a server without authentication.

- Not validated locally.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `cluster_mode` — boolean, default `true`

Connect to a Redis Cluster, with `address` listing its nodes. Set it to `false` for a single Redis server.

#### `database` — string

Number of the database RudderStack writes to. When it's omitted, RudderStack uses database `0`.

- Applies when `cluster_mode` is `false`. Leave it unset otherwise.
- Written as a string, not a number — an unquoted number fails validation.
- At most 100 characters, and must not contain line breaks. A value that isn't a number passes validation, and RudderStack then uses database `0`.

```yaml
cluster_mode: false
database: "2"
```

#### `prefix` — string

Prefix for the keys RudderStack writes: `<prefix>:user:<userId>` instead of `user:<userId>`.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

### TLS

#### `secure` — boolean, default `false`

Connect to Redis over TLS.

#### `skip_verify` — boolean, default `false`

Skip verifying the server's certificate chain and host name. That leaves the connection open to man-in-the-middle attacks, so use it only for testing, for example against a self-signed certificate.

- Applies when `secure` is `true`.

#### `ca_certificate` — string, secret

PEM-encoded CA certificate RudderStack verifies the server's certificate against. Omit it when any client can verify the server's CA, as with Amazon ElastiCache.

- Applies when `secure` is `true`. Leave it unset otherwise.
- Not validated locally.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets) for passing its line breaks.

### Storage format

By default, each user's key holds a Redis hash. RudderStack merges `context.traits` and `traits` into it, with `traits` winning, flattens nested traits with `.` — `location.city` — and stores every value as a string. Fields the event doesn't carry are left as they were.

#### `use_json_module` — boolean, default `false`

Store each user's traits as a JSON document instead of a hash. RudderStack merges each `identify` event's traits into the user's existing document.

- Needs the RedisJSON module on your Redis server.
- The dashboard shows this setting only to workspaces with the feature turned on.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Redis in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Redis accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — events reach Redis from RudderStack's servers, never in device mode.

The dashboard also accepts `warehouse` for Redis, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'redis-prod' (type 'redis') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'redis-prod' config has no 'connection_mode' entry for source type 'web'
```

Redis needs no additional config keys to connect a source of any type.

## Secrets

`password` and `ca_certificate` are the secret keys. Write each as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  password: "{{ .REDIS_PASSWORD }}"
  ca_certificate: "{{ .REDIS_CA_CERTIFICATE }}"
```

Note that:

- `ca_certificate` spans several lines. Pass it with its line breaks written as `\n`, which the double-quoted reference turns back into line breaks: `export RUDDER_REDIS_CA_CERTIFICATE="$(awk '{printf "%s\\n", $0}' ca.pem)"`.
- The dashboard doesn't mask `ca_certificate`, but Rudder CLI treats it as a secret.
