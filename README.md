# zt / zeithrold tools

`zt` is a small, cross-project tooling entry point. It discovers existing verification capabilities, distributes versioned Agent Skill directories, and will coordinate native lint packages and quality checks. Each project keeps its native build system and project-specific rules.

## Current slice

- `zt inspect`: read-only inventory. Go and JS/TS have focused detectors; Flutter, Rust, Python, and JVM have conservative initial detectors. The report separates discovered entry points from environment readiness and execution results. This command never runs checks.
- `zt plan CAPABILITY --module ID`: read-only command plan for Go and JS/TS. It prefers existing project recipes and package scripts. Generated Go fuzz commands have a 30-second budget per target.
- `zt sync --plan`: read-only preview of bundled Skill directory changes.
- `zt sync`: installs selected complete Skill directories and writes `zt.lock.json`. It refuses to overwrite unmanaged or locally edited Skill directories. Removal is intentionally manual in this slice.
- `packages/eslint-config`: unpublished JS/TS flat-config package. It is a native ESLint package, not code executed by the Go CLI.

`zt check`, `zt run`, `zt doctor`, native command planning for the other four stacks, lint distribution for other stacks, and remote releases are **not implemented yet**. This repository does not replace `ledger-tooling` or modify Ledger's current gates.

See the [Ledger assessment queue](docs/ledger-assessment.md) before extracting any existing Ledger behavior.

## Try it locally

With Go 1.24 or newer:

```sh
go test ./...
go run ./cmd/zt inspect --root /path/to/ledger --json
go run ./cmd/zt plan fuzz --root /path/to/ledger
go run ./cmd/zt inspect --root /path/to/memory
```

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

Supported stack IDs are `go`, `jvm`, `js-ts`, `flutter`, `rust`, and `python`. `expect` is currently reported by `inspect`; enforcement comes with the later `check` command. If no `zt.json` exists, `inspect` detects a manifest at the root and reports without changing the project.

The complete Skill catalog currently contains `go-testing`, `js-ts-testing`, `ui-foundation`, `ui-web`, and `ui-flutter`. Shared UI guidance describes information hierarchy, states, accessibility, reflow, and evidence. Colors, typography, component libraries, and product-specific interactions stay in each project's design contract.

## Evidence boundary

`discovered` means a test target, configuration, or command declaration was found. `not_detected` means the detector found no entry point in the files it scanned. `unknown` means the inventory was incomplete or the stack-specific semantics need inspection. `environment: not_checked` and `lastRun: not_run` remain explicit until separate commands actually probe or execute them. Warnings are advisory in this slice.

## Source layout

```text
cmd/zt/                 CLI
internal/project/       zt.json validation
internal/inspect/       read-only stack inventory
internal/skills/        Skill planning, collision detection, and sync
skills/                 complete bundled Skill directories
packages/eslint-config/ native ESLint package (unpublished)
examples/               proposed pilot configurations
```

## License and publication

License, remote visibility, release signing, and distribution channels remain open decisions. This local first slice has not been published.
