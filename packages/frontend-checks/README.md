# @ztd-me/frontend-checks

Native CSS and Playwright accessibility checks. Requires Node >=22.14 and Playwright Test ^1.62.0. Version `0.1.3` adds native Tailwind utility and text-role metadata validation, canonical string import notation, and static JS/TS utility token checks with a declared TypeScript runtime dependency, retaining the MIT license and Tailwind block custom variant fix. It requires automatic staging, owner 2FA promotion and fresh registry verification before consumers install it. See [publishing](docs/publishing.md) and [CLI integration](https://github.com/zeithrold/tools/blob/main/docs/frontend-tooling.md). Consumers retain the strict official `@ztd-me/eslint` profile and upgrade each package only after its release is verified.

After promotion and verified public installation:

```sh
pnpm add -D --save-exact @ztd-me/frontend-checks@0.1.3 @playwright/test@1.62.0
pnpm exec playwright install chromium
```

Tools verification and the three consumer projects have user approval for `minimumReleaseAgeExclude: ['@ztd-me/*']`. Preserve other packages' age gates, integrity checks, no-downgrade and build policy. No scope trust exemption is granted. The `0.0.0-stage` placeholder for a new package is not a usable helper release.

## CSS

```js
// css-check.config.mjs
export default {
  files: ['src/**/*.css'],
  // Optional: static utility colors/tokens in source.
  classFiles: ['src/**/*.{ts,tsx}'],
  tokenFiles: ['node_modules/tailwindcss/theme.css'],
  externalCustomProperties: [],
}
```

```sh
pnpm exec ztd-css ./css-check.config.mjs
```

The config module is reviewed project code and executes during import. `checkCss(options)` is also exported from `@ztd-me/frontend-checks/css`. Every supplied glob must match; missing files and syntax/configuration errors fail. JSON findings print to stdout; a nonzero exit fails the gate. Under `zt check`, the CLI also writes `css.json` to `ZT_ARTIFACTS_DIR`.

Stylelint 17.15 / standard 40 provide native CSS validation; only documented Tailwind directives and `--alpha`/`--spacing` functions receive syntax allowances. The nesting-scoping rule remains enabled, with precise Tailwind-provided roots. Version 0.1.3 additionally rechecks `@utility` and variant declarations with native property rules,
supports exact text-role metadata and canonical string import notation. Root/media nesting outside that directive still fails. See [variant compatibility](docs/tailwind-variants.md). Undefined `var()` references fail across the supplied files/declaration sources, even with fallbacks. Exact runtime-generated variables can be declared in `externalCustomProperties`; document their owner in the project contract. Imports are not resolved automatically. Neither Sass/Less nor embedded Vue styles are parsed by this CSS-only entrypoint.

Hex, named and CSS color-function paint literals outside custom-property definitions fail. Token definitions retain each project's values. `transparent`, `currentColor`, inheritance and URLs are allowed. The published declaration gate does not cover inline JS styles or every CSS color expression.
Version 0.1.3 additionally checks static string/template utility fragments when
`classFiles` is configured, including palette colors, arbitrary paint literals and undefined custom
properties. Native TypeScript AST parsing preserves comment and expression boundaries; dynamic
expressions and runtime cascade remain outside this static proof. An inventory definition is not proof of cascade/theme availability; render the relevant states. No autofix rewrites token values.

Published versions through 0.1.2 do not contain these adaptations. Consumers must upgrade to the
verified official 0.1.3 release after owner promotion; do not patch the helper, replace its CLI with a
local checkout, or add package extensions to supply its runtime dependencies. Shared checker behavior
belongs in this package and requires a new npm version. Local source and packed-consumer checks
prepare the release; successful registry verification establishes public installation.

## Playwright

```js
import { defineConfig } from '@playwright/test'
import { verificationArtifacts } from '@ztd-me/frontend-checks/playwright'

const artifacts = verificationArtifacts()
export default defineConfig({
  ...artifacts,
  webServer: existingWebServer,
  projects: existingProjects,
  use: { ...artifacts.use, baseURL: existingBaseURL },
})
```

```js
import { test } from '@playwright/test'
import { assertAccessible, captureState } from '@ztd-me/frontend-checks/playwright'

test('translated dialog', async ({ page }, info) => {
  await page.goto('/settings')
  // Establish the real dialog state using project locators and interactions.
  await assertAccessible(page, info, { label: 'settings-dialog' })
  await captureState(page, info, 'settings-dialog')
})
```

`assertAccessible` uses Axe 4.13 and defaults to WCAG 2 A/AA, 2.1 AA and 2.2 AA tags. It attaches the complete scan before asserting zero violations. Optional `include` scopes a supplemental scan; optional nonempty `tags` changes the selected coverage and must be justified locally. Full-page and keyboard/focus tests remain necessary. Projects own routes, states, browser matrices, service mocks and the built-Worker server. This helper does not provision or start them.

`verificationArtifacts(root?)` provides HTML/JSON reports, test attachments, failure traces/screenshots/videos. The root defaults to `ZT_ARTIFACTS_DIR`, otherwise `.zt/browser`. Merge with existing config; preserve native server/projects/use settings. `captureState` attaches a named PNG for review without maintaining screenshot baselines.

## Verification

```sh
pnpm install --frozen-lockfile
pnpm exec playwright install chromium
pnpm run check
```

Tests execute native CSS parsing/lint, real Chromium Axe/keyboard/dialog behavior, a deliberate accessibility failure, typed imports, fresh tarball installation/CLI use, and stage authorization/duplicate guards. The workspace uses the authorized `@ztd-me/*` age exclusion and retains the existing exact `semver@6.3.1` trust exception. Fresh consumers retain strict 24-hour release age for other packages and no-downgrade for all packages. `test:registry` verifies registry integrity against the original reviewed release tarball and repeats consumer import/type/CLI/browser pass/failure checks; metadata alone is insufficient. Source documentation updates do not overwrite an already published version.

## License

First-party code and documentation are [MIT licensed](LICENSE). Installed dependencies, including
Axe, Stylelint and Playwright, retain their upstream licenses and notices. This package does not
relicense dependencies or their transitive code. Existing published versions keep their original
artifacts and metadata.
