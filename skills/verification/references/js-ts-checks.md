# JS/TS checks that can fail

Synthetic examples only; the project's manifest, lockfile and scripts decide
what runs. Commands below illustrate the checks, not a prescribed toolchain.

## Resolve the command before trusting it

- Read the manifest script body. A `test` script that runs only a linter, or a
  runner configured to pass with no tests, supports a different claim than its
  name.
- Confirm the package manager from the lockfile and the `packageManager` field,
  then invoke that manager rather than a hand-assembled binary.
- Keep type checking, unit tests and browser tests separate; each covers a
  different claim.

```text
pnpm exec tsc --noEmit
pnpm exec vitest run src/cart/total.test.tsx
pnpm run test:browser
```

Installing dependencies is a change to project state and needs the project's
own setup rule; do not install to make a check runnable. Use the existing store
and frozen lockfile where the project already requires one.

## Build the expectation outside the component

```text
lines: two units at 1000 cents, 15% tier discount
pre-round: 2000 - 300 = 1700 cents
expected total: "17.00"
```

Write that expectation as a literal derived from the contract. Do not call the
component's own calculation to fill it, and do not accept a snapshot regenerated
with `-u` as an oracle. Snapshot suites can be useful for review, but acceptance
of changed output needs an expectation that does not come from the candidate.

For UI, assert the user-observable state and accessible name or role computed
from the specification, not values copied out of the current DOM.

## Falsify the check

In a scratch copy or a temporary local change you revert, introduce one defect
at a time:

| Perturbation | Expected failure |
| --- | --- |
| Change the quantity, route or prop | The amount, route or state assertion for that case |
| Disable the error branch | The failure and recovery test |
| Skip the outbound request or pending-state write | The request assertion (method, URL, body, count) |
| Return a cached response | The refresh or revalidation test |
| Remove an accessible name or keyboard handler | The role/name query or keyboard test |

If a perturbation stays green, the assertion is weak: it may only check that an
element exists, that a mock was defined, or that the happy path rendered. Fix
the assertion or narrow the claim; a higher test count of the same kind is not
better evidence.

For a defect, the new test should fail on the pre-change revision and pass
after. If the old cycle was not observed, label it unobserved rather than
reporting old-fail/new-pass.

## Doubles and behavior

Mock the external boundary, not the wrapper under test. A mock module, `vi.mock`
call or spy definition shows setup; it proves nothing until the test executes
the public path and asserts the interaction (arguments, call count) together
with the rendered or persisted outcome. A component stub that returns only its
fixture confirms its own fixture.

For async and stateful UI, cover pending, success, failure and retry or
cancellation states, and settle on a user-visible condition instead of an
arbitrary sleep. Advance fake timers where the behavior depends on them. Assert
outcomes rather than framework internals.

## Browser and artifact evidence

A DOM simulation cannot establish native focus, scrolling, layout, locale
rendering or a real network boundary. When the claim depends on those, run the
browser suite against the built runtime, and record the browser/channel and
version. Retain traces, reports and named captures under the project artifact
root; a screenshot or diff alone does not establish visual or accessible
approval. Leave no server, browser or watcher process running afterwards.

Read the process exit status and the per-test report (for example JSON reporter
output), not the final console summary. A CI run with a different environment
configuration is different evidence from a local run.

## Reuse and invalidation

Invalidate a result when the lockfile or package manager changes, the Node or
browser version changes, the script or test command changes, the environment or
fixture changes, or a diff touches the exercised source path. Mark carried-over
results as reused with revision and date; never present an earlier pass as this
run's evidence.
