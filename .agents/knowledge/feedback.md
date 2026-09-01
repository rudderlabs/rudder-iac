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
- Register the CLI `gcs` destination only under `ExperimentalFlags.UnverifiedDestinations`, not as a verified/native destination available unconditionally.
- Keep `s3` as the verified destination registered unconditionally; reviewer guidance explicitly corrected GCS to the unverified gate.
- Every newly onboarded destination starts under the unverified gate; promotion to verified is a separate, deliberate change after live verification.

## DEX-731 — Experimental Flag Promotion Review Guidance
<!-- ticket:DEX-731 -->
- In `docs/experimental-flags.md`, examples under "Adding a New Experimental Flag" should use placeholder flag names such as `YourNewFeature` instead of real experimental flags, so future flag promotions do not require guide rewrites.
- When removing an experimental guard around a code path that consumes configuration, check whether invalid existing user config becomes active; either preserve compatibility by clamping intentionally or return an error that names the exact config key, such as `concurrency.syncer`, so users can fix their config.
