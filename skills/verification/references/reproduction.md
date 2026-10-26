# Reproduction and process evidence

Reproduction is a claim that the reported symptom occurs under identified
conditions. It is separate from having a fix and from a test run. Examples are
synthetic and commands belong to the stack references under `references/`.

## Baseline and candidate are different builds

- Baseline: the build the reporter actually uses, such as an installed package,
  global binary or released artifact. Do not silently substitute a local
  checkout.
- Candidate: the exact local build that contains the change, built by the
  project's documented path.
- Do not edit source while establishing the baseline; a baseline obtained from a
  modified tree is a candidate.

If the installed baseline is unavailable, record it as unavailable. A local-only
reproduction is weaker evidence and must be labeled that way, not reported as
the user's case.

## Record identity

Every reproduction record names:

| Field | Content |
| --- | --- |
| Build | Version or revision, artifact digest, build flags and provenance |
| Recipe | Exact command, task target or UI steps, in order |
| Environment | OS, architecture, runtime and dependency versions, service or native component, credentials present, data class |
| Symptom | Observed versus expected, exact message, status, stack or rendered state |
| Repetition | Runs attempted, failures observed, stability statement |

A version number alone is not identity: the same version can differ by platform,
lockfile, configuration or native dependency. When baseline and candidate differ
in the environment rather than the change, the comparison does not support a fix
claim; restore the recorded environment or say the cause is unknown.

## Verdicts

- `reproduced`: the recorded symptom occurred under the recorded identity.
- `not_reproduced`: the run completed and the symptom was absent or different.
  A different symptom, or a failure caused by a missing service, permission or
  dependency, is not a reproduction of the reported problem; report the condition
  that failed instead.
- `environment_blocked`: the condition could not be reached at all.
- `unstable`: intermittent results. Record every run before and after; an
  unstable baseline cannot establish that a change fixed anything.

A wrong symptom or a blocked environment does not authorize a speculative fix
and does not establish that the defect exists elsewhere.

### Synthetic record

```text
problem: export writes an empty file for reports over 10,000 rows
baseline: cli 2.3.1 installed from release archive (digest ...), default config
candidate: local build of fix branch (revision ...)
recipe: cli export --format csv --since 2025-01-01 > out.csv
environment: linux/amd64, go toolchain from go.mod, local fixture database
symptom: out.csv is 0 bytes; expected header plus 12,431 rows
repetition: baseline 3/3 empty; candidate 3/3 complete
verdict: reproduced at 2.3.1, fixed on candidate build
```

## Repetition and stability

Run more than once when the defect is described as intermittent or when the
baseline already failed differently between attempts. State the success and
failure counts rather than a single green run. A fix claim needs the failing
baseline and the passing candidate under the same recipe; a single green run
after an unstable baseline is not a fix.

## Verify the process, session and artifact

- Read the exit status and signal, not only the last printed line.
- Inspect the session or log window the user would inspect; a session that never
  started is not a passing session.
- Open the produced artifact and check its type, schema, row count or rendered
  content. Its existence is not its correctness.
- Confirm background processes and daemons started, listened where required and
  stopped. A leftover process, port, temporary credential or changed file is a
  side effect that belongs in the report.
- A final assistant message, summary table or transcript quote is an account of
  the run. It does not replace the exit status or artifact observation.
