# Variable Substitution

Variable substitution lets you keep environment-specific values and secrets **out of your
project YAML**. Instead of hardcoding a database host or password, you write a placeholder
like `{{ .DB_HOST }}` and supply the real value at apply time from an environment variable
or a variable file.

This keeps a single set of specs reusable across `dev`, `staging`, and `prod`, and keeps
secrets out of version control.

> **Status: experimental.** This feature is behind an experimental flag and is **off by
> default**. When it is off, `{{ ... }}` in your specs is left untouched and `--var-file`
> is ignored — behaviour is identical to before. See [Enabling the feature](#enabling-the-feature).

---

## Quick start

1. **Enable the feature** (one-time setup):

   ```bash
   export RUDDERSTACK_CLI_EXPERIMENTAL=true
   export RUDDERSTACK_X_ENABLE_VAR_SUBSTITUTION=true
   ```

2. **Add placeholders to your spec** (`specs/destination.yaml`):

   ```yaml
   version: rudder/v0.1
   kind: destination
   metadata:
     name: my_postgres
   spec:
     config:
       host: "{{ .DB_HOST }}"
       port: {{ .DB_PORT | 5432 }}          # unquoted → stays a number; defaults to 5432
       database: "{{ .DB_NAME }}"
       password: "{{ .DB_PASSWORD }}"       # quote secrets and strings
   ```

3. **Create a variable file** (`staging.vars.yaml`):

   ```yaml
   DB_HOST: db.staging.example.com
   DB_NAME: analytics
   DB_PASSWORD: super-secret-value
   ```

4. **Apply with the variable file:**

   ```bash
   rudder-cli apply --location ./specs --var-file staging.vars.yaml
   ```

That's it. The CLI replaces each `{{ .VAR }}` with its resolved value *before* the YAML is
parsed, so the rest of the apply cycle sees fully-resolved specs.

---

## Enabling the feature

Variable substitution is an [experimental flag](../../../docs/experimental-flags.md) named
`enableVarSubstitution`. Two things must be true for it to take effect: experimental mode must
be on **and** the flag must be enabled.

Pick whichever method fits your workflow.

### Option A — Environment variables (great for CI)

```bash
export RUDDERSTACK_CLI_EXPERIMENTAL=true            # turn on experimental mode
export RUDDERSTACK_X_ENABLE_VAR_SUBSTITUTION=true   # turn on this feature
```

### Option B — Config file (persistent, `~/.rudder/config.json`)

```json
{
  "experimental": true,
  "flags": {
    "enableVarSubstitution": true
  }
}
```

The top-level `"experimental": true` is required — without it, all experimental flags are
ignored.

### Option C — CLI command

Once experimental mode is on (Option A or B sets `experimental`), you can toggle the flag with:

```bash
rudder-cli experimental enable enableVarSubstitution
```

---

## Syntax

### Reference a variable

```yaml
host: "{{ .DB_HOST }}"
```

- The token is `{{ .NAME }}` — the **leading dot is required** (`{{ DB_HOST }}` is invalid).
- Whitespace inside the braces is flexible: `{{.DB_HOST}}`, `{{ .DB_HOST }}`, and
  `{{  .DB_HOST  }}` are all equivalent.
- Variable names must match `[A-Za-z_][A-Za-z0-9_]*` — start with a letter or underscore,
  followed by letters, digits, or underscores. (So `{{ .123 }}` or `{{ .my-var }}` are
  invalid syntax.)
- You can put multiple tokens on one line:

  ```yaml
  url: "https://{{ .DB_HOST }}:{{ .DB_PORT }}/{{ .DB_NAME }}"
  ```

### Provide a default value

Use a pipe (`|`) to supply a fallback used only when the variable is **not** found in any
source:

```yaml
port: "{{ .DB_PORT | 5432 }}"
region: "{{ .AWS_REGION | us-east-1 }}"
```

If `DB_PORT` is defined anywhere, its value wins and the default is ignored. If it is not
defined, the default is used and no error is raised.

---

## Where values come from

Values are looked up from the following sources, in priority order. The **first** source that
provides the variable wins.

| Priority | Source | How to provide it |
| -------- | ------ | ----------------- |
| 1 (highest) | Environment variables | `export RUDDER_DB_HOST=...` |
| 2 | Variable files | `--var-file file.yaml` (later file wins over earlier — see below) |
| 3 (lowest) | Inline default | `{{ .VAR \| default }}` |

If a variable is found in none of these and has no default, the apply/validate **fails**
(see [Error handling](#error-handling)).

### Environment variables

Environment variables must be prefixed with `RUDDER_`. The prefix is stripped before matching:

```bash
export RUDDER_DB_HOST=db.prod.example.com   # resolves {{ .DB_HOST }}
export RUDDER_DB_PASSWORD=s3cret            # resolves {{ .DB_PASSWORD }}
```

Environment variables without the `RUDDER_` prefix are ignored by substitution.

### Variable files

Pass one or more files with `--var-file`. Each file is **flat key-value YAML** — top-level
keys mapped to scalar values:

```yaml
# staging.vars.yaml
DB_HOST: db.staging.example.com
DB_PORT: 5432          # numbers and booleans are allowed; coerced to strings
DB_NAME: analytics
DB_PASSWORD: "super-secret-value"
ENABLED: true
```

> **Note:** The `RUDDER_` prefix applies **only to environment variables**, not to keys in
> variable files. Use the bare variable name in YAML — `DB_HOST:`, not `RUDDER_DB_HOST:`.
> A `RUDDER_`-prefixed key in a var file will not match `{{ .DB_HOST }}` (it would be looked
> up as `{{ .RUDDER_DB_HOST }}`).

Rules for variable files:

- **Scalars only.** Strings, numbers, and booleans are allowed. Nested maps or lists are
  rejected (`DB: { HOST: x }` or `HOSTS: [a, b]` cause an error).
- **No null/empty values.** `KEY:` or `KEY: null` is rejected. To set an empty value, use
  explicit empty quotes: `KEY: ""`.
- Comments (`#`) and blank lines are fine.
- Paths are resolved relative to your current working directory.

### Combining multiple variable files

`--var-file` is repeatable. When the same key appears in more than one file, the **later
file wins**:

```bash
# values in prod.vars.yaml override the same keys in base.vars.yaml
rudder-cli apply --var-file base.vars.yaml --var-file prod.vars.yaml
```

This matches the layering convention used by `helm`, `kubectl`, `docker-compose`, and
`terraform`. Note: an environment variable (`RUDDER_*`) still wins over *any* variable file.

> The `--var-file` help text currently reads "earlier files take priority" — that text is
> stale. The actual behaviour is **later file wins**, as documented here.

---

## Supported commands

The `--var-file` flag and substitution apply to:

- `rudder-cli apply`
- `rudder-cli validate`

They are **not** available on `destroy`, `migrate`, or `import` (those commands either do not
load local specs or are out of scope for this feature). Even without `--var-file`, `RUDDER_*`
environment variables are always picked up by `apply` and `validate` when the feature is enabled.

---

## Types and quoting

Substitution happens on the raw bytes **before** YAML parsing, so the type of a value is
decided by YAML *after* substitution.

```yaml
port: {{ .DB_PORT }}      # if DB_PORT=5432, this parses as the integer 5432
port: "{{ .DB_PORT }}"    # this parses as the string "5432"
```

Recommendation:

- **Quote the reference** (`"{{ .VAR }}"`) for anything that might contain special
  characters — passwords, connection strings, URLs, or anything with `:`, `#`, `{`, `}`, etc.
  An unquoted value containing these can break YAML parsing or change the document's meaning.
- **Leave it unquoted** only when you intentionally want the value to keep its native type
  (a number or boolean from a variable file or env var).

---

## Error handling

Substitution is **strict**. If a referenced variable is not found in any source and has no
default, the command fails *before* applying anything. All problems are reported in one pass
so you can fix them together, for example:

```
error[project/var-substitution]: undefined variable "DB_PASSWORD"
  --> specs/destination.yaml:8:14
     |
   8 |   password: "{{ .DB_PASSWORD }}"
     | ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
```

| Error | Cause |
| ----- | ----- |
| `undefined variable` | `{{ .VAR }}` not found in env vars or var files, and no default given. |
| `invalid variable syntax` | Malformed token, e.g. missing dot (`{{ VAR }}`) or an invalid name (`{{ .1A }}`). |
| `variable file not found` | A `--var-file` path does not exist. |
| `failed to parse variable file` | The file is invalid YAML, has nested values, or has a null/empty value. |

When any spec has a substitution error, nothing is applied — the original specs are left
untouched.

---

## Security best practices

- **Never commit secrets.** Add secret-bearing variable files to `.gitignore` (e.g.
  `*.vars.yaml` or a dedicated `secrets/` directory). Check in only non-sensitive,
  environment-specific files if you must.
- Prefer **environment variables** (`RUDDER_*`) for secrets in CI/CD — they are not written
  to disk alongside your specs.
- The CLI never prints resolved values: error messages reference variable *names* only, never
  their values.

---

## FAQ & edge cases

**Q: What happens if I reference a variable that isn't set?**
The command fails with an `undefined variable` error pointing at the line and column. Give the
variable a value (env var or var file) or add an inline default: `{{ .VAR | fallback }}`.

**Q: How do I write a literal `{{ .X }}` that should *not* be substituted?**
There is currently **no escape syntax**. Any `{{ .NAME }}`-shaped token outside a comment is
treated as a variable. The only built-in exception is YAML comments — a token after an
unquoted `#` is left untouched:

```yaml
# TODO: wire up {{ .DB_HOST }} later   <- not substituted (it's in a comment)
host: "{{ .DB_HOST }}"                 <- substituted
```

If you genuinely need a literal `{{ .X }}` in a value, avoid the dot-prefixed pattern, or
restructure so the token sits in a comment.

**Q: Is a `#` inside a quoted string treated as a comment?**
No. A `#` only starts a comment when it is unquoted. These tokens are still substituted:

```yaml
note: "#1 priority {{ .OWNER }}"       <- substituted
label: '#hashtag {{ .CAMPAIGN }}'      <- substituted
```

**Q: Can a variable's value contain another variable?**
No. Substitution is a single pass and is **not recursive**. If a variable file sets
`B: "{{ .A }}"`, the value of `B` is the literal string `{{ .A }}` — it is not expanded.

**Q: Will my numbers stay numbers?**
Yes, if you leave the reference unquoted. `port: {{ .DB_PORT }}` with `DB_PORT=5432` parses as
the integer `5432`. Quote it (`"{{ .DB_PORT }}"`) to force a string. See
[Types and quoting](#types-and-quoting).

**Q: What does an empty value produce?**
An empty resolved value is written as an empty string (`""`) so it doesn't accidentally become
YAML `null`. In a variable file, you must write an empty value explicitly as `KEY: ""` —
`KEY:` (no value) is rejected.

**Q: Two variable files set the same key — which wins?**
The one passed **later** on the command line. And an `RUDDER_*` environment variable beats both.

**Q: Can I use nested keys or lists in a variable file?**
No. Variable files must be flat with scalar values. Nested maps/lists cause a
`failed to parse variable file` error.

**Q: Does substitution run inside embedded code blocks (e.g. a transformation's JS)?**
Yes — substitution operates on the whole file. This is usually safe because JavaScript/Python
use `${...}`, not `{{ .X }}`. But if your embedded code contains a `{{ .something }}` token, it
*will* be substituted (or error if undefined). Keep that token out of the code or put it in a
comment.

**Q: Where are `--var-file` paths resolved from?**
Relative to the directory you run the command in (your current working directory). Absolute
paths also work.

**Q: I enabled the flag but nothing happens.**
Make sure **both** the top-level experimental mode and the flag are on. With env vars that's
`RUDDERSTACK_CLI_EXPERIMENTAL=true` **and** `RUDDERSTACK_X_ENABLE_VAR_SUBSTITUTION=true`. With
the config file, `"experimental": true` **and** `"flags": { "enableVarSubstitution": true }`.

**Q: An environment variable I set isn't being picked up.**
It must be prefixed with `RUDDER_`. For `{{ .DB_HOST }}`, export `RUDDER_DB_HOST`. The prefix
is stripped during lookup.
