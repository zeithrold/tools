---
name: go-testing
description: Inspect and run a Go project's unit, fuzz, and mutation checks with clear evidence boundaries and bounded execution.
---

# Go testing

Read the project's `AGENTS.md`, `zt.json`, `go.mod`, and existing test commands before changing anything.

1. Use `zt inspect --json` to distinguish discovered test targets from tests that actually ran. Prefer the project's existing `just`, Make, or CI entry points when they define setup or isolation.
2. Run focused `go test` for the affected packages, then the project's declared full unit suite. Record the exact command and result. A passing build does not establish test coverage.
3. For a `FuzzXxx` target, run a bounded fuzz search using the project's existing command or `go test -fuzz=... -fuzztime=...` in its package. Ordinary `go test` runs the seed corpus; it does not establish that fuzz search ran.
4. Mutation testing has no single Go standard entry point. Discover the project-selected runner and target scope. Report missing tooling separately from surviving mutants and test failures.
5. Keep generated corpora and reports in project-approved locations. Preserve any reproducing fuzz input as a reviewed regression fixture.

State passed, failed, blocked, and not run separately. Do not infer production behavior from local checks.
