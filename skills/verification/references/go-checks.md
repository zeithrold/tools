# Go checks that can fail

Synthetic examples only; the project's packages, targets and toolchain still
decide what runs. Use the project entry point when one defines setup or
isolation, and keep the focused commands below as evidence of behavior rather
than substitutes for it.

## Build the expectation outside the code

```go
func TestQuoteTotal(t *testing.T) {
	tests := []struct {
		name      string
		lines     []Line
		wantCents int
	}{
		// Two units at 1000 cents with the contract's 15% tier discount:
		// 2000 - 300 = 1700, rounded half-up.
		{"tier discount", []Line{{UnitCents: 1000, Qty: 2}}, 1700},
		// Three units at 333: 999 - 149.85 = 849.15, rounds half-up to 849.
		{"half-up rounding", []Line{{UnitCents: 333, Qty: 3}}, 849},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := QuoteTotal(tt.lines); got != tt.wantCents {
				t.Fatalf("QuoteTotal = %d, want %d", got, tt.wantCents)
			}
		})
	}
}
```

The literals come from the contract or a hand calculation written down beside
them. Filling `wantCents` by calling `QuoteTotal(tt.lines)` reproduces the defect
on both sides.
Golden files regenerated with an update flag are fixture maintenance, not an
oracle; keep their source rule outside the implementation and review changes as
acceptance assets when the repository treats them that way.

Run the focused check first, then the project's declared suite:

```text
go test ./internal/quote -run '^TestQuoteTotal$' -count=1 -v
go test ./... -count=1
```

`-count=1` prevents a cached success from standing in for this run. The exit
status plus per-test `--- PASS/FAIL` lines or `-json` events are the evidence; a
later summary is not. Record `go version`, `go env GOOS GOARCH GOFLAGS`, build
tags and module state when they affect the claim.

## Falsify before trusting the check

In a scratch copy, or a temporary edit you revert before committing, introduce
these defects one at a time and confirm the test goes red:

| Perturbation | Expected failure |
| --- | --- |
| Pass the wrong quantity or wrong argument | The value assertion for that case |
| Force the other branch or skip its precondition | The rejection or boundary subtest |
| Remove the audit write, publish call or notification | Its assertion, read through the package API |
| Return a stale cached value | The re-read or freshness case |

```text
go test ./internal/quote -run '^TestQuoteTotal$' -count=1
```

Restore each perturbation after observing the red result. A perturbation that
stays green means the assertion is decorative; fix the assertion or narrow the
claim. A passing build, `go vet` or a successful type compilation is not
behavioral evidence.

For a defect, the decisive perturbation is the old revision: the new test fails
there and passes after the change. If that cycle cannot be produced, say so
rather than implying it was observed.

Mutation testing has no single Go entry point. Discover the project-selected
runner and scope, and report missing tooling separately from test failures and
surviving mutants.

## Keep the real path and fake only boundaries

- Exercise the package API and real internal dependencies; inject a clock,
  transport or store only at the external boundary needed for determinism.
- Assert that the boundary was called with the expected arguments and that the
  resulting observable changed. A defined fake or a compiled mock proves neither.
- Use `t.TempDir` for files, unique names for shared resources and `t.Setenv`
  for environment changes; avoid the user's real data and credentials.
- For concurrent code, select `-race` where the platform supports it, and
  repeat enough to catch a race. A cross-build does not establish runtime or
  credential-store behavior.
- Prefer an injected clock or event wait over sleeps; keep timeouts bounded.

## Fuzzing and retained inputs

```text
go test ./internal/parse -run=^$ -fuzz=FuzzParse -fuzztime=30s
```

An ordinary `go test` runs the seed corpus only. A bounded fuzz search is the
claim that search ran; keep it within the project's time budget, and preserve a
reproducing input under `testdata/fuzz/` as a reviewed regression rather than a
throwaway file.

## Reuse and invalidation

A Go result covers the packages, build tags, toolchain, module graph, platform
and command it ran with. Invalidate it when any of those change, when a diff
touches the exercised path, or when the test command itself changes. Label a
carried-over result as reused with its revision and date: `-count=1` applies to
a fresh run, not to a previous report.
