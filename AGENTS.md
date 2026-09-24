# Repository guidance

- Keep `zt inspect`, `zt plan`, and `zt sync --plan` read-only. They must not run project tests, install dependencies, or rewrite project files.
- Separate discovered configuration, environment readiness, and actual execution. Never report a check as passed based only on source inspection or a command name.
- Keep native toolchains in charge of linting and testing. Add only conservative adapters; ambiguous semantics should remain `unknown` with an actionable warning.
- `zt sync` owns only Skill directories explicitly selected in `zt.json` and its own lock file. Preserve project-owned Skills, `AGENTS.md`, design contracts, and local edits.
- Ledger and memory are pilot consumers. Do not modify either consumer repository while developing this CLI unless the user explicitly requests that migration step. Keep `ledger-tooling` operational during incremental evaluation.
- Run `go test ./...`, `go vet ./...`, and a local read-only inspection against the pilot checkouts after changing detectors or planning behavior. Report which checks actually ran.
