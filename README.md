# zt / zeithrold tools

`zt` is a small, cross-project tooling entry point. It discovers existing verification capabilities, distributes versioned Agent Skill directories, and will coordinate native lint packages and quality checks. Each project keeps its native build system and project-specific rules.

## Current slice

- `zt inspect`: read-only inventory. Go and JS/TS have focused detectors; Flutter, Rust, Python, and JVM have conservative initial detectors. The report separates discovered entry points from environment readiness and execution results. This command never runs checks.
- `zt plan CAPABILITY --module ID`: read-only command plan for Go and JS/TS. It prefers existing project recipes and package scripts. Generated Go fuzz commands have a 30-second budget per target.
- `zt check --module ID --profile NAME`: executes reviewed native commands in an explicit ordered profile, stops after a required failure/blocker, and retains execution reports/logs/artifacts. It never installs dependencies or deploys.
- `zt sync --plan`: read-only preview of bundled Skill directory changes.
- `zt sync`: installs selected complete Skill directories and writes `zt.lock.json`. It refuses to overwrite unmanaged or locally edited Skill directories. Removal is intentionally manual in this slice.
- `packages/frontend-checks`: publishable CSS and Playwright/Axe helpers, with packed consumer tests and an automatic stage-only npm workflow. Owner 2FA promotion and public installation verification precede consumer rollout. See [frontend integration](docs/frontend-tooling.md).
- `packages/eslint-config`: strict JS/TS/React/Vue flat-config package (`@ztd-me/eslint`), with a pnpm stage-only publication workflow. It is a native ESLint package, not code executed by the Go CLI.

`zt run`, `zt doctor`, native command planning for the other four stacks, and lint distribution for other stacks are **not implemented yet**. Release automation is configured, but no versioned release exists yet. This repository does not replace `ledger-tooling` or modify Ledger's current gates.

See the [Ledger assessment queue](docs/ledger-assessment.md) before extracting any existing Ledger behavior.

## Try it locally

With Go 1.24 or newer:

```sh
go test ./...
go run ./cmd/zt inspect --root /path/to/ledger --json
go run ./cmd/zt plan fuzz --root /path/to/ledger
go run ./cmd/zt inspect --root /path/to/memory
```

For a development install from the public repository, use Go 1.24 or newer:

```sh
go install github.com/zeithrold/tools/cmd/zt@main
zt version
```

`go install` places the binary in `GOBIN`, or `GOPATH/bin` when `GOBIN` is unset; put that directory on `PATH`. Once a release tag exists, pin it instead of using `@main`, for example `@v0.1.0`. A project should pin the same `zt` version in its CI/bootstrap recipe; `zt.lock.json` records the version and content hash of each synchronized Skill. `zt.json` does not download or update the CLI itself.

To try Skill sync, copy an [example config](examples/zt-memory.json) to `zt.json` in a **temporary project copy**, then run:

```sh
go run ./cmd/zt sync --root /path/to/project --plan
go run ./cmd/zt sync --root /path/to/project
```

`sync --plan` is safe against an existing checkout. `sync` changes only selected `.agents/skills/<id>/` directories and `zt.lock.json`; review its plan first. It never rewrites a project's own `AGENTS.md`, `DESIGN.md`, or other Skills.

## `zt.json`

```json
{
  "schemaVersion": 1,
  "modules": [
    {
      "id": "api",
      "path": ".",
      "stack": "go",
      "expect": {
        "unit": "required",
        "fuzz": "warn",
        "mutation": "warn"
      }
    }
  ],
  "skills": ["go-testing"]
}
```

Supported stack IDs are `go`, `jvm`, `js-ts`, `flutter`, `rust`, and `python`. `expect` is advisory in `inspect`; `zt check` enforces it for entries in the selected explicit profile. If no `zt.json` exists, `inspect` detects a manifest at the root and reports without changing the project.

The complete Skill catalog currently contains `go-testing`, `js-ts-testing`, `ui-foundation`, `ui-web`, `ui-flutter`, `frontend-engineering`, and `frontend-verification`. Shared UI guidance describes information hierarchy, states, accessibility, reflow, and evidence. Colors, typography, component libraries, and product-specific interactions stay in each project's design contract.

## Evidence boundary

`discovered` means a test target, configuration, or command declaration was found. `not_detected` means the detector found no entry point in the files it scanned. `unknown` means the inventory was incomplete or the stack-specific semantics need inspection. `environment: not_checked` and `lastRun: not_run` remain explicit until separate commands actually probe or execute them. Warnings are advisory in this slice.

## Source layout

```text
cmd/zt/                 CLI
internal/project/       zt.json validation
internal/inspect/       read-only stack inventory
internal/check/         native execution and evidence collection
internal/skills/        Skill planning, collision detection, and sync
skills/                 complete bundled Skill directories
packages/eslint-config/ native strict ESLint package
packages/frontend-checks/ CSS and browser verification helpers
examples/               proposed pilot configurations
```

## License and publication

The repository is public. CI tests each push and pull request. A semantic version tag triggers a GitHub Release with macOS, Linux, and Windows archives for amd64 and arm64, plus SHA-256 checksums. The Go release workflow does not publish npm packages or create tags automatically. Separate `publish-eslint.yml` and `publish-frontend.yml` workflows automatically verify relevant pushes to `main` and stage new package versions; published or pending versions are skipped. A maintainer must promote each stage with 2FA before it becomes installable. ESLint `0.1.1` is published on npm; frontend helper `0.1.0` is prepared for owner-reviewed publication. No versioned Go CLI release has been published yet. License and release signing remain open decisions.
