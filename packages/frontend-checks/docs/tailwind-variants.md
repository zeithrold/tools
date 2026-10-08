# Tailwind block variant compatibility

Version 0.1.1 fixes the valid Tailwind 4.3.3 system-dark variant rejected by frontend-checks 0.1.0:

```css
@custom-variant dark {
  :root[data-frontend-mode="dark"] & { @slot; }

  @media (prefers-color-scheme: dark) {
    :root[data-frontend-mode="system"] & { @slot; }
  }
}
```

Tailwind substitutes each utility selector for `&`. The native compiler produces the expected explicit-dark selector and system-mode media branch. Stylelint 17.15.0/standard 40.0.0 instead reports two missing-scoping-root findings because this precompiled directive is not native CSS nesting.

The fix keeps `nesting-selector-no-missing-scoping-root` enabled and uses its supported secondary option `ignoreAtRules: ['custom-variant']`. It does not ignore files, media blocks, arbitrary at-rules or other findings. Tests require root/media `&` outside custom variants to fail and preserve native invalid-property findings plus semantic literal-color and undefined-token checks. Both direct and nested-media variant roots pass. The packed CLI consumer includes the exact source variant, and native Tailwind 4.3.3 compilation is an independent regression check. This is not a claim that the source CSS checker accepts every generated utility naming convention or validates the complete Tailwind language.

The CSS API/config shape and Playwright exports are unchanged. Keep release-age, integrity, no-downgrade and existing build restrictions. Consumers need an official version containing the fix; changing tools source cannot repair an older published lock. New releases follow automatic staging and owner approval with npm 2FA. Verify the promoted exact version against its reviewed tarball using `scripts/registry-smoke.mjs` before updating final consumer locks. Staging alone is insufficient.

Reference: [Stylelint's native ignoreAtRules option](https://stylelint.io/user-guide/rules/nesting-selector-no-missing-scoping-root/#ignoreatrules).

## CSS-first validation in 0.1.3

Version 0.1.3 additionally supports nested `@utility`, native `--text-name--line-height`,
`--letter-spacing` and `--font-weight` theme metadata, and canonical string import notation. String
imports are required by Tailwind's native stylesheet loader; remote Google Fonts imports use the same
notation. Import-notation stays enabled, as do all property, nesting, color and token checks.
Stylelint treats unknown block at-rules as descriptor scopes and omits property-no-unknown there;
the adapter rechecks utility/variant declarations through the native property rule with exact original
source locations. It does not ignore unknown properties or non-Tailwind root nesting.

`classFiles` optionally names explicit JS/TS/JSX/TSX paths or globs. The native TypeScript AST checks
static string/template class fragments for palette colors, arbitrary literal colors and undefined
custom properties. Dynamic expressions remain outside this static proof; it does not claim complete
Tailwind compilation or runtime-cascade validation. Keep consumer tokens in the CSS inventory and
external runtime properties explicit. Comment text and ordinary layout utility strings are ignored.

Published versions through 0.1.2 do not contain these adaptations. Upgrade the exact official npm
version after owner promotion and fresh registry verification; consumer patches, package extensions,
and local CLI replacements are not supported delivery mechanisms. The package declares TypeScript
as a runtime dependency for AST parsing. Packed and registry consumer checks exercise the native
CLI with this contract and require deliberately invalid CSS and utility tokens to fail.
