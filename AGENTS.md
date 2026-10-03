# Repository guidance

- `tools` owns reusable generic logic only, across all packages, commands, Skills and workflows. Project-specific business logic, project-name allowlists or branches, legacy storage mappings, deployment/domain/branding/auth policies, and per-project schemas or clients belong to the owning project, not tools runtime.
- Generic mechanisms accept explicit consumer configuration or interfaces; do not add named-project exceptions. OpenAPI sharing is limited to reusable verification mechanisms: business schemas, clients, authentication and error semantics stay project-owned. Remove the rejected legacy preference mappings entirely; do not relocate them into consumers.
- Representative tests and documentation may demonstrate generic APIs with synthetic consumer configuration. They must not embed project-specific production behavior. Preserve unrelated guidance and flag other ownership violations separately rather than expanding a task into an unrequested repository rewrite.
- Keep `zt inspect`, `zt plan`, and `zt sync --plan` read-only. They must not run project tests, install dependencies, or rewrite project files.
- Separate discovered configuration, environment readiness, and actual execution. Never report a check as passed based only on source inspection or a command name.
- Keep native toolchains in charge of linting and testing. Add only conservative adapters; ambiguous semantics should remain `unknown` with an actionable warning.
- `zt sync` owns only Skill directories explicitly selected in `zt.json` and its own lock file. Preserve project-owned Skills, `AGENTS.md`, design contracts, and local edits.
- Ledger and memory are pilot consumers. Do not modify either consumer repository while developing this CLI unless the user explicitly requests that migration step. Keep `ledger-tooling` operational during incremental evaluation.
- Run `go test ./...`, `go vet ./...`, and a local read-only inspection against the pilot checkouts after changing detectors or planning behavior. Report which checks actually ran.
