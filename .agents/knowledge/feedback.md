# Feedback

> Durable human direction, corrections, and review guidance.
> Append-only. Agent-authored sections may optionally carry an HTML-comment tag
> identifying the writer/PR/run; human-authored sections are conventionally left
> untouched by automated runs.

## RUD-2963 — Optional Feature Predicate Naming
<!-- ticket:RUD-2963 -->
- Use `APIError.FeatureFlagNotEnabled()` as the public predicate name when detecting either recognized HTTP 403 feature/flag-disabled response; keep client callers and tests consistent with that name.
- Describe the two recognized API message prefixes neutrally, without assigning lifecycle labels such as legacy or GA to either one.

## DEX-608 — Destination Unverified Gate Documentation
<!-- ticket:DEX-608 -->
- When documenting or commenting on destination E2E unverified gating, do not describe HTTP as the only remaining unverified destination; the unverified registry can include multiple definitions such as `attentive_tag`, `http`, and `rs`, while S3 is verified/native.

## DEX-661 — Destination Empty Secret Export
<!-- ticket:DEX-661 -->
- Destination export should drop secret config keys before masking when the API returns the secret key with either null or an empty string value, so those keys do not become generated variable placeholders in imported YAML.

## DEX-499 — GCS Destination Gate Correction
<!-- ticket:DEX-499 -->
- Register the CLI `gcs` destination only under `ExperimentalFlags.UnverifiedDestinations`, not as a verified/native destination available with `ExperimentalFlags.DestinationSupport` alone.
- Keep `s3` as the verified destination registered with `ExperimentalFlags.DestinationSupport` alone; reviewer guidance explicitly corrected GCS to the unverified gate.
- Every newly onboarded destination starts under the unverified gate; promotion to verified is a separate, deliberate change after live verification.

## DEX-734 — Experimental Flag Promotion Review Scope
<!-- ticket:DEX-734 -->
- For experimental-flag promotion PRs, keep the diff scoped to removing the flag gate and exact stale references; do not bundle unrelated E2E fixture/snapshot reductions or new migration rewrites into the same PR.
- Preserve existing coverage during flag promotion unless the ticket explicitly asks to change it, because unrelated migration/E2E changes make the promotion harder to review and can reduce coverage.
- Hidden `rudder-cli migrate` rewrites project files in place; when validating with variable substitution enabled, the migrated write path must preserve placeholder tokens and must not write resolved `RUDDER_*` or `--var-file` secret values back to disk.
