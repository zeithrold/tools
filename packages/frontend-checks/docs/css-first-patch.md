# Local CSS-first compatibility patch

Published `@ztd-me/frontend-checks@0.1.1` remains the package base. The CSS-first
candidate has not been published. `patches/@ztd-me__frontend-checks@0.1.1.patch`
contains only the six generic CSS checker source changes. Its reviewed SHA-256 and
per-file candidate hashes are in `patches/css-first-candidate.json`.

Copy the patch and receipt into the consumer repository. Keep the exact public
package pin, native `ztd-css` command and all CSS/token/class scanning configuration.
Add these portable workspace entries; retain the consumer's existing policy:

```yaml
verifyDepsBeforeRun: error
patchedDependencies:
  '@ztd-me/frontend-checks@0.1.1': patches/@ztd-me__frontend-checks@0.1.1.patch
packageExtensions:
  '@ztd-me/frontend-checks@0.1.1':
    dependencies:
      typescript: 6.0.3
```

The package extension declares the parser dependency at its owner; a root TypeScript
installation alone is insufficient. No absolute checkout or temporary tarball path
belongs in the application manifest. Explicitly update the lockfile through native
`pnpm install`, review its changes, then run `pnpm install --frozen-lockfile` and the
consumer's unchanged native CSS gate. Checks retain the `error` readiness policy
and cannot install missing dependencies implicitly.

Run `pnpm run test:patch` in this package to install the actual public version with
the patch in a fresh synthetic consumer. The test checks candidate file hashes,
17 native CSS tests, strict typed imports, the default CSS script, three real
Chromium accessibility tests and a deliberately failing inaccessible control.
Evidence is retained in `.artifacts/css-patch-verification.json` and the referenced
temporary consumer. This verification does not promote a public package or approve
the UI source registry.

To revise the patch, prepare the exact public package with native `pnpm patch`,
copy the reviewed CSS candidate files into its edit directory, and generate the
patch with `pnpm patch-commit`. Refresh its receipt and repeat `test:patch`. Never
silently regenerate a different patch under an existing accepted hash.
