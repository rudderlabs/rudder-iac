# PR #459 Review Cache
**URL:** https://github.com/rudderlabs/rudder-iac/pull/459
**Fetched:** 2026-03-16T14:00:00Z

---

## Thread: `cli/internal/cmd/datagraph/validate/validate.go:87` (ID: PRRT_kwDONNpQU850jR5K)
**Status:** New
**Action:** Classify
**First comment DB ID:** 2939780299

### Comments
- **fxenik** (2026-03-16T11:30:53Z): As a general convention, I would prefer to avoid including business logic in the command itself and instead delegate to the validation orchestration (load project, run validations, produce output) and execution to a dedicated validator component. The purpose of the command should be to collect input (arguments) and configure the validator component directly, as well as plugging in the right dependencies (e.g project from deps. Have a look at internal/project/importer/importer.go for an example

---

## Thread: `cli/internal/cmd/datagraph/datagraph.go:10` (ID: PRRT_kwDONNpQU850jTnm)
**Status:** New
**Action:** Classify
**First comment DB ID:** 2939789354

### Comments
- **fxenik** (2026-03-16T11:32:47Z): Since data graphs are still behind an experimental flag, I would like this to be Hidden and enabled only if the experimental flag is set

---

## Thread: `cli/internal/providers/datagraph/display/validation.go:1` (ID: PRRT_kwDONNpQU850kSSG)
**Status:** New
**Action:** Classify
**First comment DB ID:** 2940125427

### Comments
- **fxenik** (2026-03-16T12:39:56Z): I would probably call this file displayer.go since this is what it implemented/provides

---

## Thread: `cli/internal/providers/datagraph/display/validation.go:43` (ID: PRRT_kwDONNpQU850kTqM)
**Status:** New
**Action:** Classify
**First comment DB ID:** 2940132728

### Comments
- **fxenik** (2026-03-16T12:41:25Z): I would prefer to have separate implementations (maybe with common input types) for different display modalities (terminal vs json). Something like a terminaldisplayer.go and jsondisplayer.go

---

## Thread: `cli/internal/providers/datagraph/display/validation.go:15` (ID: PRRT_kwDONNpQU850kVib)
**Status:** New
**Action:** Classify
**First comment DB ID:** 2940142485

### Comments
- **fxenik** (2026-03-16T12:43:18Z): Can we have line width dictated by terminal width instead of being fixed?

---

## Thread: `cli/internal/providers/datagraph/display/validation_test.go:34` (ID: PRRT_kwDONNpQU850kXLw)
**Status:** New
**Action:** Classify
**First comment DB ID:** 2940151075

### Comments
- **fxenik** (2026-03-16T12:44:56Z): Assertions for displayer output should consider the entire output, not individual lines or segment of the output. That way we would be validating padding/identation correctly. There should be one assert.Equals statement with the entire buffer contents

---

## Thread: `cli/internal/providers/datagraph/validations/planner.go:107` (ID: PRRT_kwDONNpQU850kdtZ)
**Status:** New
**Action:** Classify
**First comment DB ID:** 2940185488

### Comments
- **fxenik** (2026-03-16T12:51:47Z): I don't like how the validation unit construction is repeat across 3 plan modes. Can we have a separation of concerns where 3 plan modes are about collecting resources, and a common function converts them to validation units? Or maybe have resources to units conversion helper? Another idea is to have three separate public plan functions (PlanAll, PlanModified, PlanSingle) which also get the source graph as input, instead of having a common struct which practically shares very little between implementations, and no practical state.

---

## Thread: `cli/internal/providers/datagraph/validations/planner_test.go:65` (ID: PRRT_kwDONNpQU850kfH9)
**Status:** New
**Action:** Classify
**First comment DB ID:** 2940192963

### Comments
- **fxenik** (2026-03-16T12:53:10Z): This should not be limited to counts, we need to test against contents of the plan

---

## Thread: `cli/internal/cmd/datagraph/validate/validate.go:135` (ID: PRRT_kwDONNpQU850ko2H)
**Status:** New
**Action:** Classify
**First comment DB ID:** 2940244429

### Comments
- **fxenik** (2026-03-16T13:02:45Z): Instead of a common spinner for the entire operation, I would like to explore if I can use a pattern similar to how syncer reports tasks, where a separate spinner is shown for all validations, ideally leveraging the same underyling UI component.

---

## Thread: `cli/internal/providers/datagraph/validations/results.go:46` (ID: PRRT_kwDONNpQU850ksb5)
**Status:** New
**Action:** Classify
**First comment DB ID:** 2940263391

### Comments
- **fxenik** (2026-03-16T12:06:16Z): Let's call this ValidationReport

---

## Thread: `cli/internal/providers/datagraph/validations/runner.go:41` (ID: PRRT_kwDONNpQU850kx5f)
**Status:** New
**Action:** Classify
**First comment DB ID:** 2940292213

### Comments
- **fxenik** (2026-03-16T13:11:34Z): Instead of clumping all mode parameters in the argument list, I would prefer to have the Runner (or the Validator if you check my other comment about the command responsibility) to follow a pattern similar to options, where it gets a (mandatory) Mode struct (e.g ModelAll, ModeModified, ModeSingle{ .. }) with corresponding arguments only where they are relevant. That way we can extend to additional modes without changing the function signatures across implementation to add additional arguments.

---

## Thread: `cli/internal/providers/datagraph/validations/runner.go:95` (ID: PRRT_kwDONNpQU850k2KB)
**Status:** New
**Action:** Classify
**First comment DB ID:** 2940302464

### Comments
- **fxenik** (2026-03-16T13:13:30Z): When reporting a resource in an error, prefer using the resource's URN

---

## Thread: `cli/internal/providers/datagraph/validations/tasker.go:50` (ID: PRRT_kwDONNpQU850k2KB_tasker)
**Status:** New
**Action:** Classify
**First comment DB ID:** 2940314686

### Comments
- **fxenik** (2026-03-16T13:15:38Z): Let's include the URN in the unit and use it as a key

---
