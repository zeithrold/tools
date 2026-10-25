# Third-party licensing boundary

The root `LICENSE` covers first-party code and documentation. It does not replace upstream licenses,
copyrights, reserved font names or notices. See [the component inventory](docs/licensing.md).

| Material | Original terms and notice | Distribution |
| --- | --- | --- |
| Go runtime and standard library compiled into `zt` | BSD-3-Clause, copyright 2009 The Go Authors; `third-party/GO-BSD.txt` | Embedded in `zt license` and included in CLI archives |
| `github.com/spf13/cobra@v1.10.2` CLI command tree in `zt` | Apache-2.0, copyright The Cobra Authors; `third-party/COBRA-APACHE-2.0.txt` | Embedded in `zt license` and copied into CLI archives |
| `github.com/spf13/pflag@v1.0.9` flag parsing behind cobra | BSD-3-Clause, copyright 2012 Alex Ogier and The Go Authors; `third-party/PFLAG-BSD-3-CLAUSE.txt` | Embedded in `zt license` and copied into CLI archives |
| `github.com/inconshreveable/mousetrap@v1.1.0` Windows-only cobra helper that detects Explorer-started sessions | Apache-2.0, copyright 2022 Alan Shreve; `third-party/MOUSETRAP-APACHE-2.0.txt` | Embedded in `zt license` and copied into CLI archives |
| `github.com/99designs/keyring@v1.2.2` OS keystore abstraction `zt` uses for the macOS Keychain, Windows Credential Manager and Linux Secret Service | MIT, copyright 2015 99designs; `third-party/KEYRING-MIT.txt` | Embedded in `zt license` and copied into CLI archives |
| `github.com/99designs/go-keychain@v0.0.0-20191008050251-8e49817e8af4` cgo macOS Keychain binding that keyring compiles into darwin builds | MIT, copyright 2015 Keybase; `third-party/GO-KEYCHAIN-MIT.txt` | Embedded in `zt license` and copied into CLI archives |
| `github.com/danieljoos/wincred@v1.1.2` Windows Credential Manager binding that keyring compiles into windows builds | MIT, copyright 2014 Daniel Joos; `third-party/WINCRED-MIT.txt` | Embedded in `zt license` and copied into CLI archives |
| `github.com/gsterjov/go-libsecret@v0.0.0-20161001094733-a6f4afe4910c` Secret Service helper for keyring's Linux backends; pure Go over D-Bus, not a cgo binding to the C `libsecret` | MIT, copyright 2016 Goran Sterjov; `third-party/GO-LIBSECRET-MIT.txt` | Embedded in `zt license` and copied into CLI archives |
| `github.com/godbus/dbus@v0.0.0-20190726142602-4481cbc300e2` D-Bus transport for keyring's Linux Secret Service and KWallet backends; keyring requires the module path without `/v5` | BSD-2-Clause, copyright 2013 Georg Reinke; `third-party/GODBUS-DBUS-BSD-2-CLAUSE.txt` | Embedded in `zt license` and copied into CLI archives |
| `github.com/dvsekhvalnov/jose2go@v1.5.0` JWE code in keyring's file backend, compiled on every platform | MIT, upstream line reads `Copyright (c) 2014` with no holder name; `third-party/JOSE2GO-MIT.txt` | Embedded in `zt license` and copied into CLI archives |
| `github.com/mtibben/percent@v0.2.1` percent-encoding in keyring's file backend, compiled on every platform | MIT, copyright 2020 Michael Tibben; `third-party/PERCENT-MIT.txt` | Embedded in `zt license` and copied into CLI archives |
| `golang.org/x/sys@v0.3.0` unix system calls behind keyring's keyctl backend and `x/term` | BSD-3-Clause, copyright 2009 The Go Authors; `third-party/X-SYS-BSD-3-CLAUSE.txt` and Google's patent grant in `third-party/X-SYS-PATENTS.txt` | Embedded in `zt license` and copied into CLI archives |
| `golang.org/x/term@v0.3.0` passphrase prompt helper in keyring, compiled on every platform | BSD-3-Clause, copyright 2009 The Go Authors; `third-party/X-TERM-BSD-3-CLAUSE.txt` and `third-party/X-TERM-PATENTS.txt` | Embedded in `zt license` and copied into CLI archives |
| shadcn/ui composition | MIT, copyright 2023 shadcn; `packages/ui/third-party/SHADCN-MIT.txt` | UI source item and private packed harness |
| Radix Select 2.3.7 declaration context in the compatibility patch | MIT, copyright 2022 WorkOS; `packages/ui/third-party/RADIX-MIT.txt` | UI source item and private packed harness |
| `react-remove-scroll-bar@2.3.8` transitive Radix dependency | MIT, original upstream copyright 2025 Anton Korzunov; `packages/ui/third-party/REACT-REMOVE-SCROLL-BAR-MIT.txt` | Supplement for omitted npm notice; UI deliveries and bundled verification fixtures |
| Noto Sans | OFL-1.1, copyright 2022 The Noto Project Authors | Original `NOTO-SANS-OFL.txt` notice in UI deliveries; font binaries served by Google |
| Noto Sans SC/JP/KR | OFL-1.1, copyright 2014–2021 Adobe, Reserved Font Name `Source` | Three original CJK notices in UI deliveries; font binaries served by Google |
| Noto Emoji and Noto Color Emoji | OFL-1.1; original Google LLC / Google Inc. notices | Both original emoji notices retained; Color Emoji served by Google |

Complete UI notices remain in `packages/ui/third-party/`, with delivery-specific explanations in
`packages/ui/THIRD_PARTY_NOTICES.md` and `registry/THIRD_PARTY_NOTICES.md`. No font binaries are bundled.
OFL font software must retain OFL terms; the MIT license on the UI source does not convert the fonts.

Every Go module the `zt` build actually compiles keeps its complete original license text in the
root `third-party/` directory, byte for byte and without rewording upstream copyright lines.
`zt license` prints those embedded files and CLI archives carry the same directory, so a binary
distributed without this checkout still ships the terms that apply to it. The list follows what the
compiler and linker receive, not what `zt` selects at runtime: keyring compiles every backend into
one package per platform, so its file, prompt, Secret Service, KWallet and keyctl code is in the
binary even though `zt` only allows the OS backends.

`cobra`, `pflag` and `99designs/keyring` are on every platform. Linux adds `gsterjov/go-libsecret`
over `godbus/dbus`, plus `golang.org/x/sys` for keyctl; darwin adds the cgo `99designs/go-keychain`
binding; windows adds `danieljoos/wincred`, and `inconshreveable/mousetrap` only there. `jose2go`,
`mtibben/percent` and `golang.org/x/term` arrive through keyring's file and prompt code on all three
platforms. `go-libsecret` wraps the D-Bus protocol in Go; it does not link the C `libsecret` library,
so no LGPL library is pulled in by that name.

Cobra's own module graph also lists `go-md2man`, `blackfriday` and `yaml` for its documentation
generator; `zt` imports none of them, so no code from those modules is compiled or redistributed here.

These CLI modules are not MIT: cobra and mousetrap are Apache-2.0, `godbus/dbus` is BSD-2-Clause,
and pflag, `x/sys` and `x/term` are BSD-3-Clause. The versions named above are the ones the module
graph resolves for this change, and each file is the byte content of that version. The root MIT grant
covers first-party code only and does not relicense them.

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
[cobra](https://github.com/spf13/cobra/blob/v1.10.2/LICENSE.txt),
[pflag](https://github.com/spf13/pflag/blob/v1.0.9/LICENSE),
[mousetrap](https://github.com/inconshreveable/mousetrap/blob/v1.1.0/LICENSE),
[keyring](https://github.com/99designs/keyring/blob/v1.2.2/LICENSE),
[99designs/go-keychain](https://github.com/99designs/go-keychain/blob/8e49817e8af4/LICENSE),
[wincred](https://github.com/danieljoos/wincred/blob/v1.1.2/LICENSE),
[go-libsecret](https://github.com/gsterjov/go-libsecret/blob/a6f4afe4910c/LICENSE),
[godbus/dbus](https://github.com/godbus/dbus/blob/4481cbc300e2/LICENSE),
[jose2go](https://github.com/dvsekhvalnov/jose2go/blob/v1.5.0/LICENSE),
[percent](https://github.com/mtibben/percent/blob/v0.2.1/LICENSE),
[golang.org/x/sys](https://go.googlesource.com/sys/+/refs/tags/v0.3.0/LICENSE),
[golang.org/x/term](https://go.googlesource.com/term/+/refs/tags/v0.3.0/LICENSE),
[shadcn/ui](https://github.com/shadcn-ui/ui), [Radix](https://github.com/radix-ui/primitives),
[react-remove-scroll-bar notice](https://github.com/theKashey/react-remove-scroll-bar/blob/7301c160fda44cb8cf2b9fdfde61efad35736196/LICENSE),
[Noto Sans](https://github.com/notofonts/latin-greek-cyrillic),
[Noto CJK](https://github.com/notofonts/noto-cjk), [Noto Emoji](https://github.com/googlefonts/noto-emoji).
