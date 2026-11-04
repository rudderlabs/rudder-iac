# Custom Activation (`custom_audience`)

Custom Activation is an audience destination for an API RudderStack has no built-in integration for. You describe the HTTP calls that add, change and remove audience members, and RudderStack makes them as the rows of a Reverse ETL source change.

In a Custom Activation destination spec:

- `type: custom_audience`
- `definition_version: 1`

> [!NOTE]
> `custom_audience` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: custom-activation-prod
spec:
  id: custom-activation-prod
  display_name: Custom Activation Production
  type: custom_audience
  definition_version: 1
  enabled: true
  config:
    base_url: https://api.example.com
    authentication_enabled: true
    rudder_account_id: custom-activation-account

    headers:
      - key: X-Audience-Source
        value: rudderstack

    actions:
      insert:
        method: POST
        endpoint: /audiences/members
        request_body: '{"schema": ["EMAIL_SHA256"], "data": []}'
        batch_size: 100
        fields:
          - name: email
            is_required: true
            hash_type: SHA256
            is_custom: false
      update:
        use_insert_config: true
      delete:
        method: DELETE
        endpoint: /audiences/members
        request_body: '{"schema": ["EMAIL_SHA256"], "data": []}'
        batch_size: 100

    connection_mode:
      warehouse: cloud
```

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys and defaults.

This is the only vdm-next destination whose config is not flat: `actions` nests, exactly as the upstream payload does.

### Endpoint

#### `base_url` — string, required

Base URL every call is made against, such as `https://api.example.com`. The `endpoint` of each action is appended to it.

- Must be an `http://` or `https://` URL on a public domain, optionally with a port and a path.
- `localhost` and ngrok hosts are rejected, matching upstream. The block is anchored right after the scheme, so a host that merely *starts* with `localhost` (`https://localhost.example.com`) is rejected too.
- Templates aren't accepted in place of a literal; upstream declares the bare constraint, so a template is measured as text against it.

#### `headers` — list of objects

Extra headers sent with every call. Each entry needs a non-empty `key` and `value`.

```yaml
headers:
  - key: X-Audience-Source
    value: rudderstack
```

### Authentication

#### `authentication_enabled` — boolean

Whether the calls authenticate. Leave it out for an open endpoint.

#### `rudder_account_id` — string

The RudderStack account holding the credentials. **Required when `authentication_enabled` is `true`** and ignored otherwise:

```text
'rudder_account_id' is required when 'authentication_enabled' is 'true'
```

Create the account in the dashboard and name it here; Rudder CLI does not manage accounts, and a value naming no account in the workspace fails at apply with `Account not found with given id in the workspace`.

- At most 100 characters, and must not contain line breaks.

#### `basic_auth_user_name`, `basic_auth_password`, `bearer_token`, `api_key_name`, `api_key_value` — string

Credentials for the scheme configured on the account. Which of them apply depends on that scheme; upstream declares no constraint on any of them, so Rudder CLI validates nothing beyond their type. `basic_auth_password`, `bearer_token` and `api_key_value` are [secret keys](#secrets).

### Actions

#### `actions` — object, required

The three calls the sync makes. All of `insert`, `update` and `delete` are required.

`insert` and `delete` take the same four required keys:

| Key | Type | Notes |
| :-----| :-----| :-----|
| `method` | string, required | `POST`, `PUT`, `PATCH`, `GET` or `DELETE` |
| `endpoint` | string, required | Appended to `base_url` |
| `request_body` | string, required | Body template for the call |
| `batch_size` | integer, required | Between 1 and 5000 |
| `fields` | list of objects | The columns this call sends — see below |

`update` takes the same keys plus `use_insert_config`:

#### `actions.update.use_insert_config` — boolean

Set it to `true` to reuse the `insert` call's method, endpoint, body, batch size and fields. Then `update` needs nothing else:

```yaml
update:
  use_insert_config: true
```

Set to `false`, or left out, `update` must carry `method`, `endpoint`, `request_body` and `batch_size` of its own. Leaving the key out is not an exemption — the four are required whenever it is not `true`.

> [!WARNING]
> Rudder CLI substitutes `{{ .VAR }}` in every quoted value before it validates the spec, and it has no escape for a literal one. So a `request_body` that needs `{{ .something }}` of its own cannot be written in a spec today — `validate` reports the token as an undefined variable:
>
> ```text
> variable substitution failed: make sure undefined variables are defined in a
> variable file and passed with --var-file
> ```
>
> A body with no `{{ .x }}` token, like the sample above, is unaffected. A `{{ x }}` token without the leading dot is also left alone, since the substitution only claims the dotted form.

#### `actions.<call>.fields` — list of objects

The columns the call sends, and how each is hashed. Every entry needs all four keys:

| Key | Type | Notes |
| :-----| :-----| :-----|
| `name` | string, required | Column name |
| `is_required` | boolean, required | Whether a row without it is dropped |
| `hash_type` | string, required | `NONE`, `SHA256`, `SHA512` or `MD5` |
| `is_custom` | boolean, required | Whether it is a custom attribute rather than a standard one |

```yaml
fields:
  - name: email
    is_required: true
    hash_type: SHA256
    is_custom: false
```

### Per-source keys

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach the endpoint in. Custom Activation accepts only `warehouse: cloud`.

> [!NOTE]
> This destination models no `consent_management` key. Upstream declares no per-source-type `destConfig` lists at all, so it has no consent surface, and a spec that sets the key fails with `unknown config field`.

## Source types

| Source type | Connection mode |
| :-----| :-----|
| `warehouse` | `cloud` |

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types). Custom Activation accepts no other source type, so Reverse ETL is the only way to reach it.

## Connect a source

A Reverse ETL source connects to this destination and syncs with `sync_behaviour: mirror`:

```yaml
version: rudder/v1
kind: retl-connections
metadata:
  name: retl-connections
spec:
  connections:
    - id: audience-to-custom
      source: "#retl-source-sql-model:audience-members"
      destination: "#destination:custom-activation-prod"
      config:
        sync_behaviour: mirror
        schedule:
          type: basic
          every_minutes: 60
        identifiers:
          - from: email
            to: user_id
```

Custom Activation declares `supportsVisualMapperV2`, so the connection runs the visual mapper v2 flow: the destination names its own field set through `actions.<call>.fields`, the connection carries no `object` and no `mappings`, and `mirror` is the only sync behaviour either side accepts. Naming an `object` fails, and so does `upsert` or `full`.

Event stream sources are refused by type, since `warehouse` is the only source type this destination accepts.

## Secrets

`basic_auth_password`, `bearer_token` and `api_key_value` are the secret keys. Write each as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  bearer_token: "{{ .CUSTOM_ACTIVATION_BEARER_TOKEN }}"
```
