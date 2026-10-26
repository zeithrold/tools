# Independent oracles and falsification

An oracle decides whether an observed result is right. Status is only as
meaningful as the oracle behind it. Stack-specific commands belong to the stack
reference for the stack in use or to the project's own scripts; this file covers
how to construct and test the expectation itself. Examples are synthetic.

## Derive the expectation outside the implementation

The expected value must not come from the code path under test, its helpers, or a
fixture generated from current output. Acceptable routes:

| Route | Fits | Record |
| --- | --- | --- |
| Literal from an accepted contract or standard | Rule is small enough to state | Clause and arithmetic |
| Hand calculation in the test | A few cases cover the rule | Inputs, intermediate values, result |
| Separate property check or brute-force model | Combinatorial rule | Why it shares no logic with the candidate |
| Approved external behavior captured earlier | Compatibility claim | Source, revision, date, scope |

Snapshots regenerated from the candidate, fixtures produced by running it, and
`expected := f(input)` where `f` is under test are not oracles; they reproduce
the defect instead of exposing it.

### Synthetic example

A quote service applies a tier discount and rounds half-up to cents. The test
reads the expectation from the pricing contract, not from `QuoteTotal`:

```text
Bad:  want := QuoteTotal(lines)
      got := QuoteTotal(lines)          // both sides move together

Good: lines: two units at 1000 cents, 15% tier discount
      2000 - 300 = 1700 cents          // hand arithmetic from the contract
      want := 1700
```

For randomized inputs, record the seed and input, then derive the expected
result with a rule that shares no code with the candidate. Regenerating an
expected file with an update flag is fixture maintenance, not acceptance; an
acceptance expectation changes only under the review described in
[acceptance assets](acceptance-assets.md).

## Falsify the check before trusting it

A check that has only ever passed has unexercised discriminating power. For a new
or materially changed assertion, choose a plausible wrong result that matters to
its claim and, when feasible, confirm rejection in a disposable copy or test-only
seam. Select from the examples below only when applicable; do not require every
category, mutate protected acceptance assets, or repeat this exercise for every
use of an established unchanged gate. Do not commit the perturbation.

| Perturbation | Must fail on |
| --- | --- |
| Pass the wrong argument, quantity or route | The outcome assertion tied to that input |
| Force the other branch or bypass its precondition | The rejection, error or boundary case |
| Remove a write, notification, audit entry or request | The side-effect assertion read through the public interface |
| Return a stale or cached value | The freshness or re-read assertion |

If a relevant selected perturbation stays green, the assertion may be decorative: it checks
presence rather than value, setup rather than the call, or only the happy path.
Fix the check or narrow the claim; adding more checks of the same kind does not
help. For a defect, the strongest perturbation is the old revision: the new
check should fail there and pass on the changed revision when that cycle is
feasible. If it was not exercised, keep that evidence limit explicit rather than
calling the check useless or inventing a successful cycle.

Automated mutation testing applies the same idea at scale. Its runner and scope
belong to the stack owner; a short targeted set is often enough to judge whether
one check is load-bearing.

## Static artifacts are not behavior

- A source search for a symbol shows text, not execution.
- A defined mock, stub, spy or cleanup hook shows setup, not that anything used
  it.
- A committed snapshot shows a previously accepted rendering, not the current
  one.
- A test name or comment that claims a case is not the case.

A behavioral statement needs an executed call plus an asserted observable:
return value, emitted state, persisted row, request arguments, exit status or
artifact contents. Put doubles at the external boundary and assert the
interaction through them (arguments, count) together with the resulting outcome.

Synthetic example: a fake mail transport that records calls supports a delivery
claim only when the test runs the order path and asserts one call with the
expected recipient and body. Importing the fake, or asserting it exists, proves
nothing about delivery.
