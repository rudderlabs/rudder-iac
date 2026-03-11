---
name: apply-go-repo-conventions
description: Apply repository-specific Go code structure and quality conventions for rudder-iac. Use when writing, refactoring, or reviewing Go code to make consistent choices about package design, interfaces, context usage, error handling, concurrency, naming, and tests. Prioritize this skill whenever multiple valid Go implementations exist and you need the most idiomatic, maintainable option for this repository.
---

# Apply Go Repo Conventions

Use this skill to keep generated and reviewed Go code aligned with both:

1. `rudder-iac` local conventions
2.  Go style decisions

## Core Principles

- Prefer simplicity first: choose the smallest design that solves current requirements.
- Prefer composition over deep hierarchies or broad abstractions.
- Prefer concrete types by default; introduce small interfaces at point-of-use.
- Leverage Go's implicit interfaces for decoupling, but avoid interface-first design.
- Optimize for performance after measurement/profiling; prioritize clarity and correctness first.

## Quick Reference

Issue Type | Reference
--- | ---
Repo architecture, tests, logging, error wrapping | `references/repo-conventions.md`
Variable naming and repetition decisions | `references/naming-and-repetition.md`
Interface boundaries and abstraction choices | `references/interfaces.md`
Error wrapping, propagation, and control-flow readability | `references/error-handling.md`
Pointer/value decisions, receivers, context, testing | `references/general-go-convention.md`

## Rule Priority

When rules conflict, apply in this order:

1. Existing patterns in the touched package
2. `references/repo-conventions.md`
3. Topic-specific references (`interfaces.md`, `error-handling.md`, `naming-and-repetition.md`)
4. `references/general-go-convention.md`

## Authoring Checklist

- [ ] Package and API shape match neighboring code and avoid unnecessary exports.
- [ ] Interfaces are small and introduced at point-of-use (consumer side).
- [ ] Pointer vs value decisions are explicit and consistent for each type.
- [ ] Errors are wrapped with `%w` and include action-oriented context.
- [ ] Control flow uses early returns/continues; avoid deep `else` nesting.
- [ ] Context is first argument for request-scoped operations; do not store in structs.
- [ ] Logging is structured, useful, and does not leak secrets.
- [ ] Tests cover success, failure, and edge cases using project test conventions.
- [ ] Naming follows Go initialism conventions (`ID`, `URL`, etc.).

## Review Checklist

- [ ] Check correctness first: behavior, data flow, and lifecycle semantics.
- [ ] Check API design: exported surface, receiver choice, interface placement.
- [ ] Check safety: nil handling, error propagation, concurrency/race risks.
- [ ] Check readability: naming, repetition, short-circuit error flow.
- [ ] Check maintainability: tests are meaningful and assertions are clear.

## Implementation Workflow

1. Identify scope: API client, provider, command, syncer, or shared utility.
2. Apply package-local patterns before introducing new abstractions.
3. Run checklist before finalizing changes.
4. Validate with project commands (`make lint`, `make test`, and `make test-e2e` when apply-cycle behavior changes).

## When to Load References

- Load `references/repo-conventions.md` for all code in this repository.
- Load `references/naming-and-repetition.md` when choosing exported names, local variable names, or when reviewing repetitive naming.
- Load `references/interfaces.md` when deciding where and how to define interfaces.
- Load `references/error-handling.md` when shaping error wrapping, propagation, and branching flow.
- Load `references/general-go-convention.md` for non-topic-specific Go decisions (pointer vs value, receivers, context, testing signal quality).
