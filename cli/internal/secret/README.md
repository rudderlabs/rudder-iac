# secret

`secret.String` is a string type for sensitive values — API keys, tokens, passwords. It masks itself everywhere by default: printing, logging, formatting with any verb, and JSON/YAML marshaling all produce a masked form. The only way to get the real value out is an explicit `Reveal()` call.

The idea is to make sensitivity a property of the field's *type*. Declare a field as `secret.String` once, and it stays masked at every surface it flows through — you never have to remember to redact it by hand.

## What masking looks like

| Value | Printed / logged / marshaled as |
| --- | --- |
| `secret.New("sk-live-abc123xyz")` | `****3xyz` (last 4 characters as a spot-check hint) |
| `secret.New("short")` (under 8 characters) | `***` |
| `secret.NewUnknown()` | `(unknown)` |

## Adding a secret field to your provider

The walkthrough below uses a made-up `webhook` resource with an `apiKey` field. For a complete, compiling reference, see the example provider's book handler: `cli/internal/provider/testutils/example` (its `AccessKey` field exercises everything described here).

### 1. Spec struct: use `secret.String` (a value)

```go
type WebhookItem struct {
	ID     string        `json:"id"`
	URL    string        `json:"url"`
	APIKey secret.String `json:"apiKey"`
}
```

Users do not write the secret value in plain text — specs are committed to version control, secret values are not. Instead, the spec carries a substitution reference:

```yaml
version: rudder/v0.1
kind: webhooks
metadata:
  name: my_webhooks
spec:
  webhooks:
    - id: orders-hook
      url: https://example.com/hook
      apiKey: "{{ .ORDERS_HOOK_API_KEY }}"
```

The real value is supplied at load time, either from an environment variable with the `RUDDER_` prefix:

```sh
export RUDDER_ORDERS_HOOK_API_KEY=sk-live-abc123xyz
rudder-cli apply
```

or from a var file passed with `--var-file`:

```yaml
# secrets.vars.yaml — keep out of version control
ORDERS_HOOK_API_KEY: sk-live-abc123xyz
```

```sh
rudder-cli apply --var-file secrets.vars.yaml
```

(Substitution requires the `enableVarSubstitution` experimental flag.)

Substitution rewrites the raw spec bytes before parsing, so by the time your handler sees the spec, `apiKey` holds the real value as a bare string. No decoder setup is needed — `secret.String` knows how to decode itself from a bare string in YAML, JSON, and mapstructure, so `BaseHandler.LoadSpec` just works.

### 2. Resource (RawData) struct: use `*secret.String` (a pointer)

```go
type WebhookResource struct {
	ID     string         `json:"id"`
	URL    string         `json:"url"`
	APIKey *secret.String `json:"apiKey"`
}
```

The pointer matters: resource structs are decoded from struct to map for diffing, and only a pointer survives that round trip with its type intact. This is the same reason `*resources.PropertyRef` is a pointer.

### 3. Call `Reveal()` only where the value leaves for the API

```go
func (h *HandlerImpl) Create(ctx context.Context, data *model.WebhookResource) (*model.WebhookState, error) {
	remote, err := h.client.CreateWebhook(ctx, data.URL, revealAPIKey(data))
	// ...
}

// revealAPIKey is the single point where the real secret escapes toward the API.
func revealAPIKey(data *model.WebhookResource) string {
	if data.APIKey == nil {
		return ""
	}
	return data.APIKey.Reveal()
}
```

Outgoing API request bodies use plain string fields populated via `Reveal()`, so the real value still reaches the wire. Keep `Reveal()` confined to one small helper per handler — every call site is greppable, which is what makes revelations auditable.

### 4. Remote state: use `secret.NewUnknown()`

Backend APIs do not return secret values. When mapping a remote resource to state, mark the field as unknown:

```go
func (h *HandlerImpl) MapRemoteToState(remote *model.RemoteWebhook, urnResolver handler.URNResolver) (*model.WebhookResource, *model.WebhookState, error) {
	apiKey := secret.NewUnknown()
	resource := &model.WebhookResource{
		ID:  remote.ExternalID,
		URL: remote.URL,
		// The API never returns the real key.
		APIKey: &apiKey,
	}
	// ...
}
```

An unknown secret always diffs — even against another unknown — so the resource is re-applied on every run. That is intentional: we can never confirm the remote value matches the local one. The differ flags these diffs as secret-only and the plan output groups such resources under "always re-applied", so the user understands why the resource keeps showing up.

### 5. Export / import scaffolding: attach a variable name

When `rudder-cli import` generates specs from remote resources, a masked literal like `****3xyz` in the file would be useless — the user could never apply it. Instead, attach a substitution variable name so the marshals emit a `{{ .VAR }}` reference:

```go
func (h *HandlerImpl) MapRemoteToSpec(data map[string]*model.RemoteWebhook, inputResolver resolver.ReferenceResolver) (*export.SpecExportData[model.WebhookSpec], error) {
	// ...
	items = append(items, model.WebhookItem{
		ID:     externalID,
		URL:    res.URL,
		APIKey: secret.NewUnknown(secret.WithVariableName(apiKeyVarName(externalID))),
	})
	// ...
}

func apiKeyVarName(externalID string) string {
	return fmt.Sprintf("WEBHOOK_%s_API_KEY", strings.ToUpper(strings.ReplaceAll(externalID, "-", "_")))
}
```

Derive the name from what identifies the resource — resource type, external ID, field name — so it stays the same across re-imports. Names must match `^[A-Za-z_][A-Za-z0-9_]*$`, so fold kebab-case IDs to underscores as above. A name outside that grammar fails when the spec is generated (at marshal time), not two steps later when apply rejects the malformed token.

The generated spec then carries the reference:

```yaml
spec:
  webhooks:
    - id: orders-hook
      url: https://example.com/hook
      apiKey: {{ .WEBHOOK_ORDERS_HOOK_API_KEY }}
```

and the importer scaffolds a `secrets.vars.yaml` next to the specs with one blank entry per variable:

```yaml
# Variables referenced by the imported specs. Fill in every value before
# applying, and keep this file out of version control.
WEBHOOK_ORDERS_HOOK_API_KEY:
```

The user fills in the values and applies with:

```sh
rudder-cli apply --var-file secrets.vars.yaml
```

Values can also come from environment variables with the `RUDDER_` prefix (e.g. `RUDDER_WEBHOOK_ORDERS_HOOK_API_KEY`); environment variables take priority over var files.

Note: all of this — `WithVariableName`, the `{{ .VAR }}` output, the var file — only happens when the `enableVarSubstitution` experimental flag is on. With the flag off, `WithVariableName` is a no-op and the secret exports as a masked literal (the pre-scaffolding behaviour).

## API summary

| Function / method | Use it for |
| --- | --- |
| `secret.New(v, opts...)` | Wrapping a real, known value (spec loading does this for you) |
| `secret.NewUnknown(opts...)` | A secret whose value we never hold (remote state, export) |
| `secret.WithVariableName(name)` | Making marshals emit `{{ .name }}` instead of a mask (export) |
| `s.Reveal()` | Getting the real value — the only escape hatch |
| `s.IsZero()` | Checking there is no secret to send (empty and not unknown) |
| `s.IsUnknown()` | Checking the value is a placeholder we cannot see |
| `a.Diff(b)` | Deciding whether the secret must be (re-)applied |

## FAQ

**Why does my resource show a diff on every `apply` even though nothing changed?**
Because its remote state holds an unknown secret (`MapRemoteToState` uses `secret.NewUnknown()`), and an unknown secret always diffs. The API never tells us the remote value, so the CLI re-applies to be safe. This is by design, and the plan output classifies these resources as "always re-applied".

**Will `fmt.Printf("%q", s)` or a log statement leak the value?**
No. `String`, `Format`, `GoString`, and `LogValue` are all implemented, so every `fmt` verb (`%v`, `%s`, `%q`, `%x`, `%+v`, `%#v`), `log`, and `slog` produce the masked form. JSON and YAML marshaling mask too.

**How do I get the real value?**
`s.Reveal()`. It is deliberately the only way, so `grep -rn "Reveal()"` finds every place a real secret is used.

**Why `secret.String` in the spec struct but `*secret.String` in the resource struct?**
Spec structs are decoded once from YAML, where a value works fine. Resource structs additionally go through a struct→map→struct round trip for diffing, and a value field does not survive it: `secret.String` has no exported fields, so decoding the struct into a map turns the secret into an empty `map[string]any{}`. On the reverse conversion there is nothing left to recognise — an empty map carries no hint that it used to be a secret — so the value and the field's secret behaviour are lost. A pointer is carried through as-is in both directions, type intact. In the future, a custom decode hook could recognise `secret.String` and take care of this conversion itself; once that exists, both `secret.String` and `*secret.String` will work in resource structs.We do something very similar for PropertyRef too.

**Do I need a mapstructure hook or a custom YAML decoder?**
No. `secret.String` implements `UnmarshalYAML`, `UnmarshalJSON`, and `UnmarshalMapstructure`, so every existing decode path in the codebase accepts it without extra wiring.

**What does the user actually write in their YAML?**
A substitution reference (`apiKey: "{{ .MY_KEY }}"`) — never the value in plain text, since specs are committed to version control. The real value is resolved at load time from a `RUDDER_`-prefixed environment variable (`RUDDER_MY_KEY`) or a var file passed with `--var-file`. Substitution requires the `enableVarSubstitution` experimental flag.

**What does `Reveal()` return on an unknown secret?**
The empty string — there is no real value to reveal. Use `IsUnknown()` to tell "no secret" apart from "a secret we cannot see".

**How do I avoid sending an empty secret to the API?**
Check `IsZero()` before including the field in a request. `secret.New("")` equals the zero value, so an empty secret is treated the same as "unset".

**Why did my export emit `****3xyz` instead of `{{ .VAR }}`?**
Either no variable name was attached (`MapRemoteToSpec` did not use `WithVariableName`), or the `enableVarSubstitution` experimental flag is off — the option is a no-op without it, since a reference could never be resolved on apply.

**Why did export fail with "does not satisfy the variable grammar"?**
The name passed to `WithVariableName` does not match `^[A-Za-z_][A-Za-z0-9_]*$`. Names often derive from user-controlled external IDs, so sanitize them (e.g. uppercase and replace `-` with `_`) before attaching.

**Can destination `SecretKeys` point to nested config values?**
Yes. Destination definitions keep `SecretKeys []string`, but the destination helper layer also treats entries as dotted map paths such as `s3.access_key_id`. Top-level keys continue to work. Dotted paths are resolved manually in `cli/internal/providers/destination/configpath` so `*secret.String` values are not JSON round-tripped away; empty path segments and numeric array-index segments are rejected because slice secrets are out of scope. During export, dots become underscores in variable names, so destination external ID `my-dest` and secret path `s3.access_key_id` produce `MY_DEST_S3_ACCESS_KEY_ID`.

**Is the masked value ever sent to the API?**
No. Marshaling only masks specs and exports. Request bodies are built from plain string fields you populate explicitly with `Reveal()`, so what reaches the wire is the real value — or nothing, if you skip the field via `IsZero()`.

**Can a secret live inside a slice in my resource struct?**
It still masks and still re-applies, but the differ compares slices whole, so the diff is not flagged as secret-only and the plan renders it as a normal masked update instead of "always re-applied". Prefer top-level or map-nested secret fields until secret-awareness is extended to slices.
