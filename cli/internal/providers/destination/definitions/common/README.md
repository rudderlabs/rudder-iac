# Consent Management

Shared helpers for destination `consent_management` config: local↔API property
mapping and canonical validation.

## YAML shape

`consent_management` is keyed by **local** CLI source types (snake_case), not
API keys. Each source type maps to an array of consent entries.

```yaml
config:
  consent_management:
    web:
      - provider: oneTrust
        consents:
          - analytics
          - marketing
    react_native:
      - provider: custom
        resolution_strategy: and
        consents:
          - "{{ .CONSENT_CATEGORY || analytics }}"
          - env.CONSENT_CATEGORY
    ios_swift:
      - provider: ketch
        consents:
          - essential
```

Only source types listed on the destination definition are allowed. Local keys
are converted to API keys via `LocalToAPISourceTypes()` (for example
`react_native` → `reactnative`).

## Consent entry fields

| Field | Required | Notes |
| --- | --- | --- |
| `provider` | yes | One of `custom`, `iubenda`, `ketch`, `oneTrust` |
| `resolution_strategy` | when `provider` is `custom` | One of `and`, `or` |
| `consents` | no | Array of consent category strings |

Within one source-type array, each `provider` may appear at most once.

## Consent value rules

Each consent string must be at most 100 characters, or use one of these
pass-through forms (accepted even when longer than 100 characters):

- environment reference: `env.VAR_NAME`
- UI-style template with fallback: `{{ path || fallback }}`

Invalid plain values report: `'consent' must be at most 100 characters`.
