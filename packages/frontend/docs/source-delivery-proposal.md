# @ztd-me/ui source delivery decision

Status: the owner approved the compact chrome and source direction and authorized branch publication
with an English draft PR. Merge remains with the owner. A fresh public install after merge is required
before consumer migrations. No updater, synchronization service or production migration is included.

## Selected implementation

Deliver generic UI source as `@ztd-me/ui` through a shadcn-compatible registry in tools. Each
consumer owns the installed files and accepts updates through a reviewed source diff. Keep provider,
context and hook together in one source item; keep each Radix root and its parts on the same compatible
dependency graph. Source delivery improves visibility and local ownership, but does not automatically
deduplicate React or Radix.

The item contains the UI, provider/context/hook, required preference logic, tokens, motion, Google Fonts imports
and license notices. Installed source has no `@ztd-me/frontend` dependency. No renamed npm UI binary or
headless runtime package is proposed. The existing verification helper remains a separate npm tool.
Existing exported symbol names remain compatible to keep the source migration bounded.

The root `registry.json` inventory and generated `registry/ui.json` implement namespace `@ztd-me`, item
`ui`, and title `@ztd-me/ui`. The source currently uses the legacy package directory as its canonical
upstream location during transition. Consumers receive local files under their configured `aliases.ui`;
there is no second separately maintained source copy. See the [exact installation recipe](../../../registry/README.md).
The namespace alias uses a pinned raw GitHub item URL; the direct GitHub address is
`zeithrold/tools/ui#<SOURCE_SHA>`. Neither route installs an npm package named `@ztd-me/ui`.

## Models considered before the owner decision

| Concern | Current npm UI | Source registry | Small pure runtime + registry UI |
| --- | --- | --- | --- |
| Updates | Version bump; dependency diff | Reviewed upstream source diff | Runtime bump plus reviewed UI diff |
| Local ownership | Wrappers and token overrides; internals upstream-owned | Consumer owns and edits installed files | Consumer owns UI; pure logic upstream-owned |
| Drift | Local overrides may mask changes | Local edits need explicit merge decisions | UI drift remains; logic behavior stays versioned |
| Reproducibility | Exact version, pnpm lock and integrity | Pinned item/dependency commits, CLI version and lock | Both source and package pins |
| Shared tokens | Central package CSS changes on bump | Versioned token source; local adapters preserved | Same token ownership as source registry |
| Fixes | Central runtime patch plus consumer upgrade | Reviewed source patch per consumer | Central logic fix; UI fixes still reviewed |
| Migration effort | None | Import/provider/style replacement and verification | Same UI migration plus a new boundary |
| Identity risk | Incompatible dependency/peer graphs can duplicate contexts | Same risk unless dependencies align | Keep React context source-owned; pure logic has no identity dependency |

The table records the tradeoffs considered; source registry delivery is selected. The Radix duplication
suspicion is not established by the isolated audit. Missing menu CSS motion is
reproduced with a single compatible graph. Choosing source delivery is therefore an ownership and update
decision, not a claimed repair for a proven duplicate-instance defect.

## Dependency and update policy

- Pin the selected shadcn CLI version, every registry item to a full commit SHA, and every cross-item
dependency to its own full SHA. Pin compatible React and Radix direct dependencies in the consumer's
  package manifest and commit its native pnpm lock. Verify peer ranges before installation.
- Keep provider/context/hook atomic. Do not mix installed UI pieces with a second package's provider or
  another release's Radix root/parts. Audit `pnpm why` and resolved module identities when composition
  changes; a successful source copy alone is insufficient.
- Deliver the exact declaration-only Select 2.3.7 patch with an explicit pnpm workspace recipe. Strict
  source checking reproduces upstream TS2320 without it and passes with it; no library-check relaxation
  or JavaScript patch is needed. Preserve consumer security settings and unrelated patches.
- Inspect `add --dry-run`, `--diff` or `--view` first. Stage upstream source in a temporary directory;
  review the diff, reconcile local edits, then apply an explicit consumer-owned commit. Do not overwrite
  business files or regenerate them automatically.
- Record the upstream item SHA, CLI version and locally owned file paths in the consumer's existing
  documentation. Git and the pnpm lock provide the audit trail; no new updater service is proposed.
- Share generic tokens, component mechanisms, licenses, Skills and verification contracts. Consumers
  retain branding, domains, persistence policy, navigation, authentication, business data and deployment.
  Existing Skill synchronization does not gain ownership of UI files or consumer `AGENTS.md`.
- Run the consumer's native lint/CSS/types/build and interaction checks after each accepted update.
  Include SSR/theme first paint, keyboard/focus/inert, reduced motion, responsive captures and Axe.

The official [GitHub registry route](https://ui.shadcn.com/docs/registry/github) supports a root
`registry.json` in an existing repository, full-SHA item references and inspection before installation;
it does not require a registry server. Dependency references do not inherit the parent reference, so
each must be pinned. The [item schema](https://ui.shadcn.com/docs/registry/registry-item-json) describes
the delivered files and dependency requirements. CLI behavior is documented in the
[official reference](https://ui.shadcn.com/docs/cli).

## Bounded migration sequence

1. Submit the approved chrome and source direction, Noto typography, exact inventory, compatibility
   patch and native/browser evidence through the authorized draft PR. The owner reviews and merges.
2. After merge, verify a fresh public install with the pinned CLI and full approved source SHA through
   `node scripts/source-smoke.mjs <SOURCE_SHA>` from that exact checkout.
3. Dispatch website, showcase and memory as separate consumer changes. Replace npm UI imports, provider
   and stylesheet atomically, align dependencies and commit the lock. Preserve project-owned edits,
   branding and policy; audit action/navigation/status icons for Lucide usage, accessible names,
   decorative SVG semantics and consistent sizing. Preserve prose, mathematics, user content and
   intentional Noto Color Emoji. Run each consumer's native gates and obtain owner acceptance for each.
4. Keep the old installed npm package available until each source replacement is validated. Retiring its
   repository harness or applying npm deprecation is a separately reviewed operation; do not unpublish
   or abruptly delete it. Blog and Workbench implementation waits until the four priority repositories
   meet the owner's acceptance criteria.

Workbench remains outside this implementation: it cannot depend on `@ztd-me/frontend`, and registry
adoption there has not been approved. Do not include it automatically in these rollouts. The
documented GitHub registry integration does not support GitHub Enterprise hosts; any approved enterprise
source/vendor delivery route requires its own decision, dependency audit and native verification.
