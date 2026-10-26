# Deliberately defective review fixture

This synthetic module is test input, not a production example to adopt. It has
three known contract defects and a passing public happy-path test. Root Go tests
do not enter this nested module.

```sh
go test ./...                     # expected: pass
go test -tags acceptance ./...    # expected: fail three regression checks
```

Run from this directory. `acceptance_test.go` is the independent oracle and must
not be given to a trial reviewer; neither should the sibling `review-rubric.json`
or this explanation. Trial reviewers receive only `service.go`,
`service_test.go`, and the review task. A fresh context alone does not enforce
that restriction; record whether isolation was real or instruction-only.

Use identical source revisions and task budgets for each condition. Compare
concrete findings and unsupported claims, not prose length or number of roles.
Store responses, source hashes and judge reports in ignored artifacts. A single
fixture screens a review method; it cannot establish broad effectiveness.
