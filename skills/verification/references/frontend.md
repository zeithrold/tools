# Web frontend verification

Read the [JS/TS checks](js-ts.md), the project's design contract and native scripts. Use the `ui-design` Skill for design and experience decisions and `frontend-engineering` for architecture, preferences and data boundaries. Skill guidance does not itself run checks.

This reference covers Web/JS/TS gates, not every frontend runtime. For Flutter,
use the project's native analyze/widget/integration entry points and platform
evidence requirements; do not translate the pnpm/Worker sequence onto a native
application. A platform simulator and an actual device support different claims.

1. Inspect modules/script contents using zt inspect/plan. Confirm pnpm and browser/Worker preconditions; keep discovery read-only.
2. Define an ordered explicit profile in zt.json. Follow [gate integration](frontend-gates.md). Required absent/failed checks must fail; do not relabel them as warnings to obtain a pass.
3. Run strict JS/TS lint, CSS validation, compiler, focused unit/integration tests and the native build. Preserve @ztd-me/eslint limits independently of existing project violations.
4. Run browser interactions/Axe against the intended built runtime. Follow [evidence](frontend-evidence.md). Distinguish real services, mocked APIs and native fallback coverage.
   For persistence or async changes, exercise restore/migrate/reset, delayed responses
   and recovery as applicable. For streaming/cache changes, prove meaningful content
   and correct freshness after mutations or identity changes, not merely a fast shell.
5. Retain reports, CSS findings, failure logs, traces and named captures under the artifact root. Upload on CI success/failure; unretained /tmp captures do not fulfill this contract.
6. Report revision, tool/package versions, commands, scope, results, artifact paths and blocked/manual work. Visual baseline management remains deferred. Apply explicitly approved fixture budgets; consumer performance budgets remain consumer-owned.

Verify action/navigation/status icons use Lucide components rather than Unicode/emoji substitutes.
Check accessible names on icon-only controls, `aria-hidden="true"` on decorative SVGs, and consistent
size/stroke/alignment by role. Preserve prose, mathematics, user content and intentional Noto Color
Emoji; keep composed emoji specimens in the rendering gate. Include this audit in consumer migrations.

The shared UI harness has owner-approved combined font/API CSS response budgets: English cold
500,000 bytes, Chinese cold 1,000,000 bytes, and each immediate warm reload 10,000 bytes. Cold scenarios
use separate fresh browser contexts; warm reloads use the same context with default caching. Report
font/CSS and encoded/decoded sizes separately. The multilingual/emoji specimen retains rendering,
weight, sequence, CSP and request-count checks plus transfer reporting without an ordinary-page byte
cap. Local-font preview cannot verify remote budgets; actual Google API CI and public installation must.

## Shared shell boundary

For shared-source acceptance, require a full owner-approved source SHA and fresh
public installation of that exact source; local packed/source evidence is not
that gate. Keep upstream licenses and actual Google Fonts verification intact.
The optional `frontend-engineering` Skill or the project's component
documentation owns shared-shell implementation, consumer configuration and
delivery details. No sibling installation is required for this evidence method.
