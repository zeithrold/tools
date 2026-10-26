# Protected acceptance assets

Acceptance assets answer "does this behave as accepted?" independently of the
implementation. When one change can edit both the behavior and the answer,
acceptance collapses into self-report. Examples are synthetic.

## What is protected

- The acceptance recipe: environment setup, ordered steps and required services.
- The driver or harness that invokes product entry points.
- Fixtures, seed data, hidden cases and their versions.
- Expected results, thresholds, assertions and failure classifications.
- The known-defect reproduction list and its classification.

Protected means the change under acceptance does not edit them to make outcomes
agree. It does not mean the assets are frozen forever; they change on their own
review path.

## What stays allowed

- Ordinary unit, integration and regression tests beside the implementation,
  including a new regression for the defect being fixed.
- Test refactors that keep the same expectations and coverage.
- New fixtures for the implementer's own tests.
- Reporting a suspected acceptance-asset bug with evidence.

An incorrect protected asset changes in a separate change with its own review,
reason and version pin. A deliberate behavior change follows the same path:
the expected result updates in the reviewed acceptance change, not as an
implementation-side adjustment to make the candidate green. The acceptance run
then uses the reviewed version; the candidate does not carry the adjustment.

## Signs of adaptation

| Change | Why it is adaptation |
| --- | --- |
| Expected value edited in the same change as the implementation | The oracle now encodes the candidate |
| Fixture rows removed so a migration check passes | The case no longer exercises the reported data |
| Driver skips the step failing in the candidate | The failure becomes invisible instead of fixed |
| Threshold lowered after the failure was seen | The gate is narrowed to fit the result |
| Hidden case deleted as "unused" | Coverage disappears without a decision record |

Synthetic example: a role-migration driver expects legacy `owner` rows to be
preserved. The candidate edits the fixture to drop those rows. The correct
response is to fix the migration, or to file a separate reviewed asset bug with
evidence that the expectation is wrong. An ordinary unit test near the migration
code may add coverage at any time.

## Record the asset side

Name which protected assets were read, at which revision or digest, which ran,
and whether any changed. Treat an asset change during implementation as a risk
that needs independent review, never as evidence of success. Acceptance whose
assets were edited by the implementer is unverified.
