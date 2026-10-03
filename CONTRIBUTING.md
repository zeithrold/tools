# Contributing

## Branch and commit names

Use one focused branch from current `main` for each new agent contribution:

```text
<type>/<scope>-<description>
```

Format agent-authored commits and PR titles as:

```text
<type>(<scope>): <imperative summary>
```

Allowed types are `feat` (new behavior), `fix` (bug), `refactor` (internal restructuring),
`perf` (performance), `docs` (documentation), `test` (test coverage), `style` (formatting only),
`build` (dependencies/toolchain), `ci` (workflows) and `chore` (other maintenance).

Types, scopes and branch descriptions use English lowercase kebab-case. Choose a short scope for the affected
repository area: for example `cli`, `eslint`, `frontend-checks`, `ui`, `skills` or `repo` for shared
repository guidance. These are examples, not an exhaustive list. For a change spanning areas, use the
main affected area or `repo`; keep the subject focused on the actual result. The branch and PR title
describe the overall contribution; each commit describes its own change, so a `fix/ui-…` branch may
also contain a `test(ui): …` commit.

| Branch | Commit subject / PR title |
| --- | --- |
| `fix/ui-select-focus` | `fix(ui): restore Select focus after dismissal` |
| `feat/cli-json-inspection` | `feat(cli): add JSON inspection output` |
| `ci/frontend-checks-node-matrix` | `ci(frontend-checks): verify supported Node versions` |
| `docs/repo-agent-contribution-naming` | `docs(repo): define agent branch and commit naming` |

Write the summary in English, start with an imperative verb, omit a trailing period, and keep the full
subject at most 72 characters. Use a body when the reason, compatibility impact or validation needs
explanation. Avoid vague descriptions such as `updates`, `misc` or `final`, and agent names, timestamps
or rollout-stage labels. If a branch name is already in use, make the description more specific.
For an intentional breaking API change, use the conventional `!` marker after the scope and explain
the impact and migration in a `BREAKING CHANGE:` footer; this does not automatically request a release.

Continue an existing contribution on its assigned branch. These rules apply to new agent branches and
future agent-authored commits; they do not authorize rewriting published history, renaming active
remote branches or changing Git-generated merge/revert messages. Explicit owner instructions prevail.

## Documentation and verification evidence

Keep `docs/` and package documentation focused on maintained contracts, architecture, installation,
migration and troubleshooting. Update links when moving or removing a document. Preserve unrelated
content and all required license notices.

Put disposable test and review evidence in ignored `.artifacts/`, `.zt/artifacts/` or `artifacts/`
directories. This includes screenshots, videos, traces, run-specific JSON measurements, receipts and
comparison reports. Keep reusable test fixtures in their test directories. Link CI runs and retained
artifacts in the PR description; do not copy their output into `docs/reviews/` or package documentation.
The existing browser and source-install scripts already write to artifact directories, which CI
uploads on success or failure.

`packages/ui` owns the canonical UI source and its private verification harness. Keep
`packages/frontend-checks` separate; it remains a published verification helper. Do not publish the UI
harness to npm. The existing `@ztd-me/frontend@0.2.0` artifact remains available.

When changing source registry documentation or files, regenerate the payload from the repository root:

```sh
pnpm dlx shadcn@4.21.1 build registry.json --output registry
```

Review the generated diff and run the native package checks with pnpm. A source revision requires a
fresh public installation with its full approved SHA before consumer acceptance; local or preview
receipts do not establish that gate. Keep credentials and security configuration out of evidence.
