# Go test strategy

Go's own toolchain runs the checks; the Skill owns the claim, the statuses and
the evidence contract, and this file carries the Go commands and limits. Read
the project's `AGENTS.md`, `zt.json`, `go.mod` and existing test commands before
changing anything. Focused commands, synthetic table examples and the
perturbations that must fail are in [checks that can fail](go-checks.md); read
it when writing or judging a regression, and when a script name could stand in
for behavior.

1. Use `zt inspect --json` to distinguish discovered test targets from tests that
   actually ran. Prefer the project's existing `just`, Make or CI entry points
   when they define setup or isolation.
2. Run focused `go test` for the affected packages, then the project's declared
   full unit suite. Record the exact command and result. A passing build does not
   establish test coverage.
3. For a `FuzzXxx` target, run a bounded fuzz search using the project's existing
   command or `go test -fuzz=... -fuzztime=...` in its package. Ordinary
   `go test` runs the seed corpus; it does not establish that fuzz search ran.
4. Mutation testing has no single Go standard entry point. Discover the
   project-selected runner and target scope. Report missing tooling separately
   from surviving mutants and test failures.
5. Keep generated corpora and reports in project-approved locations. Preserve
   any reproducing fuzz input as a reviewed regression fixture.

Test the behavior through the real package API. Keep the implementation and its
actual dependencies in the test path; fake only the external boundary needed for
determinism (for example a transport or clock), not the behavior being checked.
For a defect, add a focused regression and, when feasible, observe it fail before
the fix and pass after it. Label an unobserved old-revision result as unrun.

Derive expected values outside the implementation under test, use `-count=1`
when claiming a fresh run (a cached `ok` is not a new run), and read the exit
status and per-test result rather than a later summary. Reuse a result only for
the same package scope, toolchain, build tags, platform, module graph and
command, and label it reused.

When synchronization or native integration changes, select the relevant race and
platform checks explicitly. A cross-build does not establish native runtime or
credential-store behavior. Use isolated test resources, not production data or
the user's credentials, and report a missing native environment as blocked.

State passed, failed, blocked and not run separately. Do not infer production
behavior from local checks.
