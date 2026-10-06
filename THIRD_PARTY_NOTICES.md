# Third-party licensing boundary

The root `LICENSE` covers first-party code and documentation. It does not replace upstream licenses,
copyrights, reserved font names or notices. See [the component inventory](docs/licensing.md).

| Material | Original terms and notice | Distribution |
| --- | --- | --- |
| Go runtime and standard library compiled into `zt` | BSD-3-Clause, copyright 2009 The Go Authors; `third-party/GO-BSD.txt` | Embedded in `zt license` and included in CLI archives |
| shadcn/ui composition | MIT, copyright 2023 shadcn; `packages/ui/third-party/SHADCN-MIT.txt` | UI source item and private packed harness |
| Radix Select 2.3.7 declaration context in the compatibility patch | MIT, copyright 2022 WorkOS; `packages/ui/third-party/RADIX-MIT.txt` | UI source item and private packed harness |
| `react-remove-scroll-bar@2.3.8` transitive Radix dependency | MIT, original upstream copyright 2025 Anton Korzunov; `packages/ui/third-party/REACT-REMOVE-SCROLL-BAR-MIT.txt` | Supplement for omitted npm notice; UI deliveries and bundled verification fixtures |
| Noto Sans | OFL-1.1, copyright 2022 The Noto Project Authors | Original `NOTO-SANS-OFL.txt` notice in UI deliveries; font binaries served by Google |
| Noto Sans SC/JP/KR | OFL-1.1, copyright 2014–2021 Adobe, Reserved Font Name `Source` | Three original CJK notices in UI deliveries; font binaries served by Google |
| Noto Emoji and Noto Color Emoji | OFL-1.1; original Google LLC / Google Inc. notices | Both original emoji notices retained; Color Emoji served by Google |

Complete UI notices remain in `packages/ui/third-party/`, with delivery-specific explanations in
`packages/ui/THIRD_PARTY_NOTICES.md` and `registry/THIRD_PARTY_NOTICES.md`. No font binaries are bundled.
OFL font software must retain OFL terms; the MIT license on the UI source does not convert the fonts.

Ordinary npm dependencies are installed separately, rather than vendored or bundled into first-party
package output. Private UI verification fixtures do bundle runtime dependencies; their Vite builds
retain the full original license texts in `dist/client/THIRD_PARTY_LICENSES.json` beside the bundle.
Their own package licenses and notices govern their code, including transitive
dependencies. In particular, the current ESLint package depends on `eslint-plugin-sonarjs@4.2.2`
(LGPL-3.0-only), frontend checks use `axe-core@4.13.0` through `@axe-core/playwright` (MPL-2.0), and
UI source installs `lucide-react@1.49.0` (ISC). These are upstream metadata identifiers, not a claim
that the dependency graph is MIT. Consumers distributing or bundling dependency code must preserve
and comply with its actual notices and terms.

The owner can apply MIT to first-party work. That grant does not remove upstream licensing conditions
or attribution requirements. A requirement for exclusively MIT third-party terms needs a separate owner
decision: retain the current terms and notices, obtain any additional permissions needed, or approve
specific replacements. OFL font software remains under OFL; the upstream MPL and LGPL obligations
also remain unchanged. This change retains current functionality and upstream attributions.

Original sources: [Go](https://go.googlesource.com/go/+/refs/tags/go1.24.13/LICENSE),
[shadcn/ui](https://github.com/shadcn-ui/ui), [Radix](https://github.com/radix-ui/primitives),
[react-remove-scroll-bar notice](https://github.com/theKashey/react-remove-scroll-bar/blob/7301c160fda44cb8cf2b9fdfde61efad35736196/LICENSE),
[Noto Sans](https://github.com/notofonts/latin-greek-cyrillic),
[Noto CJK](https://github.com/notofonts/noto-cjk), [Noto Emoji](https://github.com/googlefonts/noto-emoji).
