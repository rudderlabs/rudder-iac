# Destination E2E Tests

Destination onboarding includes happy-path apply lifecycle coverage under
`TestDestinationsApply`. This complements `definition_test.go`: unit tests stay
exhaustive for validation, conversion, unknown-key, mutual-exclusion, consent,
and round-trip behavior; the e2e proves the live apply → update → re-apply flow
for the destination variations that exercise distinct backend-facing paths.

## Coverage principle

Cover each destination's meaningful config variations, not every field
permutation. A meaningful variation is a distinct behavior path such as:

- each supported auth mode;
- mutually-exclusive config branches;
- alternate payload shapes that the backend handles differently.

S3 is the worked example:

- `s3.yaml` covers key-based auth (`role_based_auth: false`) with access keys
  supplied via `{{ .S3_ACCESS_KEY_ID }}` and `{{ .S3_ACCESS_KEY }}`;
- `s3-role.yaml` covers role-based auth (`role_based_auth: true`) with
  `iam_role_arn`.

Both variations also participate in the create → update mutation: the update
fixture changes `prefix`, toggles `enable_sse`, and changes display names so the
live update path is exercised.

**Never mutate an immutable key in the update fixture.** `schema.json` marks some
keys `"rs-immutable": true`, and the backend rejects any change to one with a
400:

```
Field "underscoreDivideNumbers" is immutable and cannot be modified
```

Warehouse destinations carry several — postgres marks `underscoreDivideNumbers`
and `allowUsersContextTraits`; bq adds `namespace`, `partitionColumn`,
`partitionType` and `skipViews`. Before writing the update fixture, list them
from the destination's `schema.json`: the flag sits on each entry under
`configSchema.properties.<key>`.

Every immutable key must be byte-identical between the create and update
fixtures; pick a mutable key (a bucket name, prefix, sync frequency, display
name) to exercise the update path instead. The CLI does not model immutability,
so nothing catches this before the live run — a fixture that toggles one only
fails during the gated e2e.

Do not duplicate unit-test exhaustiveness here. Keep edge cases and invalid
configs in `definition_test.go`; keep e2e focused on successful lifecycle paths.

## Fixture layout

Destination specs use a catalog-style layout so all destination e2e fixtures are
applied together with no test-code changes:

```text
cli/tests/testdata/destinations/
├── create/
│   ├── <variation>.yaml
│   └── <variation-2>.yaml
├── update/
│   ├── <variation>.yaml
│   └── <variation-2>.yaml
└── destinations.vars.yaml
```

`TestDestinationsApply` points `apply -l` at `testdata/destinations/create` and
then `testdata/destinations/update`. The project loader walks those directories
recursively, so adding a destination e2e means dropping new YAML files into both
folders and adding matching snapshots.

Keep create/update fixture filenames aligned for readability, but the fixture
file name is not the resource identity. The stable identity is `spec.id`.

## Vars and secrets

Put all destination e2e variables in the single merged var file:

```text
cli/tests/testdata/destinations/destinations.vars.yaml
```

Use template placeholders for sensitive or write-only values in YAML:

```yaml
access_key_id: "{{ .S3_ACCESS_KEY_ID }}"
access_key: "{{ .S3_ACCESS_KEY }}"
```

The committed var values must be dummy, non-production values. Never commit real
credentials. `TestDestinationsApply` asserts that raw values from the var file do
not appear in apply or destroy output; update that secret assertion if a new
fixture introduces additional literal secret placeholders.

## Snapshot layout and naming

Expected upstream snapshots live beside the destination scenario:

```text
cli/tests/testdata/expected/upstream/destinations/
├── create/
│   └── destination_<id>
└── update/
    └── destination_<id>
```

`DestinationSnapshotTester` fetches all CLI-managed destinations from the live
stack, filters them by `externalId`, count-checks them against the expected file
set, and compares each payload. Snapshot filenames are derived from the managed
URN:

1. fixture `spec.id: <id>`
2. managed URN `destination:<id>`
3. snapshot file `destination_<id>`

Concrete S3 example:

- `create/s3-role.yaml` has `spec.id: s3-role`;
- the managed URN is `destination:s3-role`;
- the expected create snapshot is
  `cli/tests/testdata/expected/upstream/destinations/create/destination_s3-role`;
- the expected update snapshot is
  `cli/tests/testdata/expected/upstream/destinations/update/destination_s3-role`.

Snapshots should include the full upstream response shape that the API returns.
Volatile fields are still present, but set to placeholders because the test's
ignore list ignores their values, not their presence:

```json
{
  "id": "<ignored>",
  "workspaceId": "<ignored>",
  "version": "<ignored>",
  "externalId": "<id>",
  "name": "E2E <Destination>",
  "type": "<APIType>",
  "enabled": true,
  "config": {},
  "createdAt": "<ignored>",
  "updatedAt": "<ignored>"
}
```

## Capturing or updating snapshots

A live destination-enabled stack is required to capture authoritative snapshots.
The test is gated to avoid accidental workspace cleanup or unsupported-destination
failures:

```bash
RUN_DESTINATION_E2E=1 go test ./cli/tests -run TestDestinationsApply -count=1 -v
```

Prerequisites:

- a valid RudderStack CLI config or `RUDDERSTACK_ACCESS_TOKEN` for the target
  stack;
- destination APIs available for the stack;
- unverified destinations enabled when any fixture uses an unverified type;

Recommended workflow:

1. Add create/update fixtures for every meaningful variation.
2. Add or update dummy variables in `destinations.vars.yaml`.
3. Run the gated e2e against a clean/live test workspace.
4. Fetch the CLI-managed destinations from the public API (the same `/v2/destinations`
   list used by `DestinationSnapshotTester`) and keep only entries with
   `externalId` set to the fixture IDs.
5. Copy each response into the matching create/update snapshot file, replacing
   volatile field values with `"<ignored>"` while keeping those keys present.
6. Re-run the gated e2e until create, update, and re-apply pass.
7. Also run the ungated compile/skip check:

   ```bash
   go test ./cli/tests -run TestDestinationsApply -count=1
   ```

If no live stack is available during onboarding, still add the fixtures and
snapshots when they can be derived safely; otherwise document the e2e deferral in
the final report with the reason and the meaningful variations that still need
coverage.

## Current limitations

The destination e2e is happy-path lifecycle coverage only. Follow-up fixtures may
cover additional S3 behavior such as `consent_management` and transformation
linking, but those are not required by the current S3 catalog-layout baseline.
