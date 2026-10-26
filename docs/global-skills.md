# Global Skills and project knowledge

## Ownership

Reusable agent workflows and the design, testing and verification Skills ship in
this repository's [`skills/`](../skills/) directory and install to the current
user's `~/.agents/skills/<id>/`. A project's `.agents/skills/` keeps project
knowledge only: business semantics, architecture boundaries, repository-specific
tools and processes. Project `AGENTS.md` files and design contracts remain the
local authority and are not overridden by a global Skill.

A global install means the Skill exists where hosts that read that path can
discover it. It does not guarantee that every host discovers the directory, and
only names and trigger descriptions are indexed; a body is loaded per task.
Verify host support before depending on a Skill, and do not copy a second tree
for an unsupported host, which only creates drift.

## Bundled Skills

Run `zt skills list` (or `--json`) for the catalog embedded in the current binary.
It reads no configuration, home directory or network.

| Task | Guidance |
| --- | --- |
| Nontrivial cross-boundary work | `task-planning` |
| Evidence, Go/JS tests and frontend gates | `verification` with relevant native references |
| Independent change assessment | `code-review` |
| Reusable instruction changes | `skill-authoring` |
| UI principles and Web/Flutter experience | `ui-design` with relevant platform references |
| Frontend architecture, SSR and state | `frontend-engineering` |
| Server boundaries, persistence and idempotency | `backend-engineering` |
| Unknown-cause defects | `systematic-debugging` |
| Independent delegated work | `agent-team` |
| Measured workload improvements | `performance-engineering` |
| Evidence-based learning proposals | `evolution-proposals` |

Select only relevant guidance. Workflows do not authorize actions or require
loading every neighboring Skill. Reusable behavioral screening cases
live in [workflow-evaluation.json](../internal/skills/testdata/workflow-evaluation.json);
they require independent trials and are not an automated model-quality gate.

### Catalog, installation and activation are different

Keep an entry point for a coherent, independently requested job. Language,
platform, specialist lenses, long examples and failure cases belong in conditional
references under that owner, not a new Skill merely because the topic has a name.
There is no target directory count or universal optimum installation size.

Each selected Skill must carry all required instruction resources within its own
directory. Optional cross-Skill routing is not a hidden installation dependency:
mentions of another owner do not require loading it, and sync never installs
siblings implicitly. Declare real external tool requirements separately.

At task time, load the entry point and only references needed for the current
stage. Do not routinely load every domain/workflow. Review and team coordination
remain different jobs; one must not recurse into the other just because their
documents mention each other.

This packaging policy follows progressive disclosure in the
[Agent Skills specification](https://agentskills.io/specification) and
[coherent-work guidance](https://agentskills.io/skill-creation/best-practices.md).
Descriptions and near-miss activation tests matter independently of body length;
see [description guidance](https://agentskills.io/skill-creation/optimizing-descriptions.md).
Host invocation controls are not universal: [Codex](https://developers.openai.com/codex/skills/)
and [Claude Code](https://code.claude.com/docs/en/skills) have distinct extensions.
No supported host is assumed to discover or load every installed file identically.

## Command contract

`zt skills sync` is the canonical command; `zt sync` is an alias with the same
flags and behavior. Both distribute to the user-global tree only and never read a
project `zt.json`, so the result does not depend on the working directory.

```text
zt skills sync --skills verification,ui-design --plan
zt skills sync --skills verification,ui-design
zt skills sync --plan
zt skills sync
```

- `--plan` reports installs, updates, restores, adoptions, unchanged directories
  and conflicts without creating a directory, lock, guard or cache. It installs
  no dependency, runs no test and makes no network request.
- `--skills` is a comma-separated selection for this run only; it never rewrites
  the user configuration. An explicitly empty `--skills` is an error rather than
  a fallback to the configuration. Invalid or duplicate ids are rejected.
- `--json` prints the plan or result as one machine-readable document.
- `--global` restates the default so the destination is explicit on the command
  line.
- `--root` is rejected whenever it is present, including `--root=`, so old
  project-scoped automation cannot silently turn into a global write. Migrate
  project copies deliberately, by hand.

## Selection

Without `--skills`, `sync` reads `~/.agents/zt.json`. The schema is strict —
`schemaVersion` and `skills`, nothing else — and must select at least one Skill:

```json
{
  "schemaVersion": 1,
  "skills": ["verification", "ui-design"]
}
```

A missing or empty selection is an error that explains how to create the file.
`zt` never installs every bundled Skill by default and never expands a selection
on its own. A project `zt.json` keeps module and native-command configuration
only; it does not carry a global Skill selection.

## What sync writes

- Whole Skill directories are copied — `SKILL.md`, `references/`, `scripts/`,
  license files and everything else — with content digests recorded in
  `~/.agents/zt.lock.json`. Only the selected `~/.agents/skills/<id>/`
  directories and that lock are managed.
- Actions are `install`, `update`, `restore`, `adopt` and `unchanged`. An
  existing directory `zt` does not manage, or a managed directory whose content
  no longer matches the lock, is reported as `conflict` and aborts the write
  instead of being overwritten.
- A previously managed Skill that is no longer selected produces a warning. `zt`
  never deletes it automatically.
- Apply plans again while holding a guard lock, then stages each directory and
  verifies the replaced copy, restoring the previous tree on failure. Content
  `zt` cannot prove it wrote is kept, and symlinked path components are refused.

## Retired entry points

| Retired ID | New owner and complete content |
| --- | --- |
| `go-testing` | `verification/references/go.md` and `go-checks.md` |
| `js-ts-testing` | `verification/references/js-ts.md` and `js-ts-checks.md` |
| `frontend-verification` | `verification/references/frontend.md`, `frontend-gates.md`, `frontend-evidence.md` |
| `ui-foundation` | `ui-design` principles and design-contract/decision/matrix/visual references |
| `ui-web` | `ui-design/references/web.md` and `web-browser-review.md` |
| `ui-flutter` | `ui-design/references/flutter.md` |

Update explicit selections deliberately; retired IDs are not aliases that install
a new Skill silently. Install and verify the selected new owners first. Compare
old managed copies and preserve local-only changes before deleting specifically
authorized retired directories and their lock records. Existing project copies,
personal unmanaged Skills and unrelated directories remain untouched. This is a
scoped migration, not a new automatic deletion behavior of `zt sync`.

## Scope

A Skill grants no network, public-write, keystore or deployment permission; its
instructions stay advisory, and loading several Skills adds no capability.
Remote GitHub writes are governed by the GitHub App and keystore contract in
[GitHub App and system keystore](github-app-setup.md), not by this distribution.

## Choosing scope at task time

- Questions and mechanical documentation edits: use only the context needed; do
  not default to planning, multiple agents or full test runs.
- Localized fixes: domain knowledge plus targeted regression tests; add debugging
  when the cause is unknown.
- Cross-module work: settle interfaces and a plan first, and parallelize only
  genuinely independent parts.
- High-risk authorization, migration or concurrency work: add independent review
  and evidence from the real entry point; urgency does not skip those gates.
- Read project knowledge only in its project or domain. Do not promote brands,
  business schemas, production environments or secrets into global rules.
