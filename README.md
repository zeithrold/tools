# zt / zeithrold tools

`zt` is a small, cross-project tooling entry point. It discovers existing verification capabilities, plans and runs reviewed native checks, distributes versioned Agent Skill directories, and prepares reviewed Skill evolution candidates. Each project keeps its native build system and project-specific rules.

## Current slice

- `zt inspect`: read-only inventory. Go and JS/TS have focused detectors; Flutter, Rust, Python, and JVM have conservative initial detectors. The report separates discovered entry points from environment readiness and execution results. This command never runs checks.
- `zt plan CAPABILITY --module ID`: read-only command plan for Go and JS/TS. It prefers existing project recipes and package scripts. Generated Go fuzz commands have a 30-second budget per target.
- `zt check --module ID --profile NAME`: executes reviewed native commands in an explicit ordered profile, stops after a required failure/blocker, and retains execution reports/logs/artifacts. It never installs dependencies or deploys.
- `zt evolve prepare` / `zt evolve plan`: prepare immutable local Skill replacement candidates and revalidate their pinned policy/base without applying them, running checks or contacting GitHub. See [offline candidate format](docs/evolve.md).
- [v0.2.0](https://github.com/zeithrold/tools/releases/tag/v0.2.0) adds `zt evolve check`, `submit` and `status`: explicit trusted-local verification, exact public-payload approval and a recoverable Draft PR workflow. Shipping the commands is not acceptance evidence; `check` runs with the operator's own authority rather than in a sandbox. Live submission and independent-review claims require their own run-specific evidence. See the [lifecycle contract](docs/evolve.md).
- `zt sync --plan`: read-only preview of user-global Skill directory changes.
- `zt sync [--global] [--skills ID[,ID]]`: installs complete Skill directories into `~/.agents/skills/<id>` and records them in `~/.agents/zt.lock.json`. The selection is the `skills` list of the user-global `~/.agents/zt.json` or an explicit `--skills` list; zt never selects the whole catalog by default, and it refuses to overwrite unmanaged or locally edited Skill directories. Removal is intentionally manual in this slice.
- `zt skills list` / `zt skills sync`: discover bundled Skills and use the same global distributor. `zt sync` remains an alias; neither manages project knowledge.
- `zt auth login/status/logout`: GitHub App Device Flow with `99designs/keyring` and only the current OS's native credential backend. Login requests expiring user tokens by default; local status does not contact GitHub unless `--probe` is explicit. See [GitHub App setup](docs/github-app-setup.md).
- `zt completion`: Cobra-generated shell completion. Commands support help and explicit JSON output; non-TTY runs do not automatically open a browser.
- `zt license`: prints the embedded first-party license and complete upstream CLI dependency notices, including in standalone binaries.
- `packages/frontend-checks`: publishable CSS and Playwright/Axe helpers, with packed consumer tests and an automatic stage-only npm workflow. Owner 2FA promotion and public installation verification precede consumer rollout. See [frontend integration](docs/frontend-tooling.md).
- `packages/eslint-config`: strict JS/TS/React/Vue flat-config package (`@ztd-me/eslint`), with a pnpm stage-only publication workflow. It is a native ESLint package, not code executed by the Go CLI.
- `packages/ui`: canonical editable UI source and private verification harness; no npm UI publication.
- `registry.json` and `registry/`: `@ztd-me/ui` source delivery through the existing GitHub route. Consumers own installed source and reviewed updates; see the [registry recipe](registry/README.md). The owner reviews upstream changes; fresh public installation is required before consumer migrations.

`zt doctor`, evolve activation, native command planning for the other four stacks, and lint distribution for other stacks are **not implemented**. The [roadmap](docs/roadmap.md) separates what the releases provide, what still needs acceptance evidence, and what has only been drafted. `zt run` is deliberately not planned. A local fixture workflow does not establish successful live GitHub submission. [v0.1.0](https://github.com/zeithrold/tools/releases/tag/v0.1.0) was the first versioned CLI release and [v0.2.0](https://github.com/zeithrold/tools/releases/tag/v0.2.0) is the current one. This repository does not replace `ledger-tooling` or modify Ledger's current gates.

See [contributor guidance](CONTRIBUTING.md) for maintained documentation, generated registry files and CI evidence.
See [Release and mise distribution](docs/distribution.md) for native platform gates, checksums,
installation/rollback, and removing an old mise Go-backend installation before a formal switch.

## Try it locally

With Go 1.26 or newer (required by the authenticated Git fetch dependency):

```sh
go test ./...
go run ./cmd/zt inspect --root /path/to/ledger --json
go run ./cmd/zt plan fuzz --root /path/to/ledger
go run ./cmd/zt inspect --root /path/to/memory
```

macOS native Keychain support uses the dependency's CGO binding and needs the Apple compiler/SDK.
Linux uses Secret Service over D-Bus; Windows uses Credential Manager. There are no handwritten OS
bindings in zt and no plaintext credential fallback. Linux's upstream keyring package may initialize a
session-bus connection (and discover/start a bus) at process startup; inventory commands do not call
credential methods, authenticate, run project checks or write project files.

For a pinned Release install with mise:

```sh
mise use -g 'github:zeithrold/tools[matching=zt_]@0.2.0'
zt version
```

If an older mise Go-backend installation already provides `zt`, follow the
[replacement procedure](docs/distribution.md#replacing-an-existing-installation)
to remove its global request and installation after verifying the replacement.
Upgrading the binary does not synchronize global Skills.

Alternatively, build the tagged source with Go 1.26 or newer:

```sh
go install github.com/zeithrold/tools/cmd/zt@v0.2.0
zt version
```

`go install` places the binary in `GOBIN`, or `GOPATH/bin` when `GOBIN` is unset; put that directory on `PATH`. Use `@main` only when deliberately following unreleased development. Skill distribution is user-global, so pin the same `zt` version where machines are provisioned; `~/.agents/zt.lock.json` records the version and content hash of each synchronized Skill. Neither config file downloads or updates the CLI itself.

Release builds embed the public GitHub App Client ID from `ZEITHROLD_GITHUB_CLIENT_ID`; development
builds can supply `zt auth login --client-id CLIENT_ID`. This is not a token flag. Device Flow uses a
Client ID, not the numeric App ID; no client private key or client secret is distributed.

To try Skill distribution, copy the [example global config](examples/zt-global.json) to `~/.agents/zt.json`, or select Skills for a single run:

```sh
go run ./cmd/zt sync --plan
go run ./cmd/zt sync --skills ui-design,verification
go run ./cmd/zt sync
```

`sync --plan` is read-only: it reports installs, updates, adoptions and conflicts without creating directories or a lock. `sync` writes only `~/.agents/skills/<id>/` and `~/.agents/zt.lock.json`. It never reads a project `zt.json`, so the result is identical from any working directory, and it never touches project `.agents/skills`, `AGENTS.md`, or `DESIGN.md`. Skills, project knowledge, and design contracts that live in a repository stay project-owned; copy a Skill into the global tree only as an explicit, reviewed distribution decision. See [global Skills](docs/global-skills.md).

`zt sync --root DIR` is rejected with an explanation. Project-level Skill sync was removed; an existing project `.agents/skills/` tree and its `zt.lock.json` are left untouched for you to review and migrate by hand.

## `zt.json`

A project `zt.json` declares modules and their native expectations:

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
  ]
}
```

Skill selection has a separate config at `~/.agents/zt.json`. It contains only `schemaVersion` and `skills`; project sections such as `modules` and `lint` are rejected there, and a project `zt.json` no longer distributes Skills:

```json
{
  "schemaVersion": 1,
  "skills": ["verification"]
}
```

`--skills ID[,ID]` overrides that list for a single run and never rewrites the config file.

Supported stack IDs are `go`, `jvm`, `js-ts`, `flutter`, `rust`, and `python`. `expect` is advisory in `inspect`; `zt check` enforces it for entries in the selected explicit profile. If no `zt.json` exists, `inspect` detects a manifest at the root and reports without changing the project.

Run `zt skills list` for the embedded catalog. Native Go/JS/frontend testing is part of `verification`; Web/Flutter design guidance is part of `ui-design`. Other entries own distinct workflows and frontend/backend decisions; see [selection and ownership](docs/global-skills.md#bundled-skills) and the [research adoption map](docs/skill-adoption.md). Catalog size, installed selection and task-time loading are separate: load only what the current task and stage need. Colors, typography, component libraries, and product-specific interactions stay in each project's design contract.

## Evidence boundary

`discovered` means a test target, configuration, or command declaration was found. `not_detected` means the detector found no entry point in the files it scanned. `unknown` means the inventory was incomplete or the stack-specific semantics need inspection. `environment: not_checked` and `lastRun: not_run` remain explicit until separate commands actually probe or execute them. Warnings are advisory in this slice.

## Source layout

```text
cmd/zt/                 CLI
internal/project/       zt.json validation
internal/inspect/       read-only stack inventory
internal/check/         native execution and evidence collection
internal/evolve/        Skill candidates, trusted-local verification and reviewed Draft PR submission
internal/skills/        user-global Skill planning, collision detection, and sync
internal/githubauth/    GitHub App device authorization and credential lifecycle
internal/keystore/      restricted native keyring adapter and process coordination
skills/                 complete bundled Skill directories
packages/eslint-config/ native strict ESLint package
packages/frontend-checks/ CSS and browser verification helpers
packages/ui/            editable UI registry source and private verification harness
examples/               example project and user-global configurations
```

## License and publication

All first-party code and documentation in this checkout are licensed under [MIT](LICENSE), including the Go CLI, npm packages, UI registry source and templates, Skills and workflows. Package archives and copied Skill directories include the full license with the existing Zeithrold copyright notice. Preserve original third-party notices; see the [licensing inventory](docs/licensing.md) and [third-party boundary](THIRD_PARTY_NOTICES.md). Dependencies and Noto fonts retain their upstream terms.

The repository is public. CI tests each push and pull request. A semantic version tag triggers native macOS, Linux and Windows release jobs for amd64 and arm64, with real credential-store gates, SHA-256 checksums and GitHub provenance attestations. Defining these jobs does not establish that they ran for a particular tag; see the distribution contract for actual acceptance requirements and the published release for its assets. CLI archives retain first-party and upstream dependency notices. The Go release workflow does not publish npm packages or create tags automatically. Separate `publish-eslint.yml` and `publish-frontend.yml` workflows automatically verify relevant pushes to `main` and stage new package versions; published or pending versions are skipped. A maintainer must promote each stage with 2FA before it becomes installable. Existing published npm versions retain their original artifacts and metadata; licensing changes apply to this checkout and future distributions.
