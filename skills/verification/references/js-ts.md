# JS/TS test strategy

The project's package manager and declared scripts run the checks; the Skill
owns the claim, statuses and evidence contract, and this file carries the JS/TS
commands and limits. Read the project's `AGENTS.md`, package manifest, lockfile
and `zt.json` before running commands. Focused commands, synthetic examples and
the perturbations that must fail are in [checks that can fail](js-ts-checks.md);
read it when writing or judging a regression, and when a script name or snapshot
update could stand in for behavior.

1. Use `zt inspect --json` to find declared scripts. Confirm the package manager
   from the project's lockfile and `packageManager` field.
2. Run the narrowest relevant lint, typecheck and test scripts. A script name is
   an entry point, not proof of what its underlying command covers; inspect it
   when the distinction matters.
3. Separate unit tests from browser or end-to-end tests. Record required
   browsers, services and credentials as preconditions; report unavailable
   infrastructure as blocked.
4. Verify affected UI states and failure recovery with appropriate tests and
   captures. Do not claim visual approval from a screenshot diff or a passing DOM
   assertion alone.

Exercise the real public API, component and SDK integration; mock transport or
other external boundaries rather than replacing the wrapper under test with a
stub that can only confirm its own fixtures. Add a focused regression for a
defect and observe old-fail/new-pass when feasible; state when the old behavior
was not reproduced.

Derive expected amounts, labels and request bodies outside the component; never
regenerate a snapshot and call it acceptance. Read the exit status and per-test
report, not the final summary. Reuse a result only for the same lockfile,
runtime, browser, command and environment, and label it reused.

For async or stateful behavior, cover relevant pending, success, failure and
retry/cancellation states through the interface users or callers actually use.
Keep assertions on outcomes, not framework internals. Check native browser
behavior when DOM simulation cannot establish it; a type check or build is not a
substitute for that evidence.

Report exact commands, results and unrun checks. Keep project-specific framework
and deployment steps in the project.

For frontend aggregate gates, use [frontend verification](frontend.md) and an
explicit `zt check` profile. Include CSS and build checks, retain browser
reports/attachments, and preserve required failures. A successful aggregate exit
does not establish untested routes or states.
