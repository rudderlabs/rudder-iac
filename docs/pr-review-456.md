# PR #456 Review Cache
**URL:** https://github.com/rudderlabs/rudder-iac/pull/456
**Fetched:** 2026-03-12T14:10:00Z

---

## Thread: `cli/internal/app/dependencies.go:161` (ID: PRRT_kwDONNpQU85z2xnS)
**Status:** Auto-approved (pending implementation)
**Action:** Implement
**First comment DB ID:** 2924219917

### Comments
- **fxenik** (2026-03-12T12:14:33Z): It feels like the account name resolver is an internal detail of the data graph provider, while the Accounts client is the external dependency. Does it make sense to create the resolver internally?

### Plan/Decision
**What:** Move AccountNameResolver creation inside the provider, accepting the AccountGetter dependency instead.

**How:**
- In `cli/internal/providers/datagraph/provider.go`, change `NewProvider` to accept `datagraph.AccountGetter` instead of `datagraph.AccountNameResolver`, and call `datagraph.NewAccountNameResolver` internally
- In `cli/internal/app/dependencies.go`, pass `c.Accounts` directly to `dgProvider.NewProvider` and remove the `dgHandlerPkg` import
- Update test calls in `cli/internal/providers/datagraph/provider_test.go` to pass `nil` (or a mock `AccountGetter`) instead of `nil` `AccountNameResolver`

---

## Thread: `cli/internal/providers/datagraph/handlers/datagraph/handler.go:109` (ID: PRRT_kwDONNpQU85z26Eg)
**Status:** Auto-approved (pending implementation)
**Action:** Implement
**First comment DB ID:** 2924265881

### Comments
- **fxenik** (2026-03-12T12:23:04Z): Instead of failing if name cannot be resolved, let's fallback to a default `data-graph` and have the namer deduplicate if needed. That is to say, maybe here we are not throwing an error and we return an empty value instead, have the formatter decide on defaults

### Plan/Decision
**What:** Change LoadImportableResources to gracefully handle account name resolution failures by leaving AccountName empty, letting the formatter/namer decide on defaults.

**How:**
- In `cli/internal/providers/datagraph/handlers/datagraph/handler.go`: Replace the error return in LoadImportableResources with a continue — when GetAccountName fails, skip setting AccountName (leave it empty)
- In `cli/internal/providers/datagraph/handlers/datagraph/handler_test.go`: Update the "returns error when account resolution fails" test to assert that resolution proceeds with empty AccountName values
- In `cli/internal/providers/datagraph/handlers/datagraph/accounts.go`: Change accountNameResolver.GetAccountName to return ("", nil) instead of an error when both name and definition type are empty
- In `cli/internal/providers/datagraph/handlers/datagraph/accounts_test.go`: Update the "returns error when both name and definition type are empty" test to expect ("", nil)

---

## Thread: `cli/internal/providers/datagraph/provider_test.go:722` (ID: PRRT_kwDONNpQU85z2_zd)
**Status:** Auto-approved (pending implementation)
**Action:** Implement
**First comment DB ID:** 2924297284

### Comments
- **fxenik** (2026-03-12T12:28:17Z): Let's have all export tests in their own file to avoid this one getting too big

### Plan/Decision
**What:** Move all FormatForExport tests and helpers into a dedicated test file.

**How:**
- Create `cli/internal/providers/datagraph/export_test.go` with the `buildRemoteResources` helper and all `TestFormatForExport_*` test functions
- Remove those functions from `cli/internal/providers/datagraph/provider_test.go`

---

## Thread: `cli/internal/providers/datagraph/provider.go:322` (ID: PRRT_kwDONNpQU85z3CA8)
**Status:** Auto-approved (pending implementation)
**Action:** Implement
**First comment DB ID:** 2924309528

### Comments
- **fxenik** (2026-03-12T12:30:24Z): This is a very big function, let's try to split to reduce cognitive load, e.g one part groups relevant resources, another formats a single data graph spec. Look for other opportunities as well

### Plan/Decision
**What:** Split the large FormatForExport method into smaller, focused helper methods.

**How:**
- Extract a `groupResourcesByDataGraph` method that builds the lookup maps (modelsByDG, relsByKey, modelExternalIDs) and returns them
- Extract a `formatDataGraphSpec` method that takes a single RemoteResource (data graph) plus the lookup maps and produces a single FormattableEntity
- Extract a `buildInlineModelSpecs` method that takes a data graph's models and relationships and returns []ModelSpec plus import resource entries
- Keep FormatForExport as a slim orchestrator: group resources, sort DGs, iterate calling formatDataGraphSpec, collect results

---
