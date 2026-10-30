# License declarations

This is the **manually maintained** licensing index. Keep declarations,
attribution and upstream links here, not copies of complete license texts.
There is no license-document generator. Dependency or distribution changes
require a human review of this index and the necessary retained notices.

## First-party work

First-party code and documentation are MIT, copyright 2026 Zeithrold:
see the canonical [LICENSE](../LICENSE). Package metadata and Skill frontmatter
declare `MIT`. That grant does not relicense dependencies.

The full MIT notice in each npm package, standalone Skill directory and UI
source delivery is intentional: those units are copied independently of this
checkout. Keep it with copies and adaptations. Do not add license copies to
ordinary source directories that are not independent distribution units.

## Adopted Skill guidance

[`prompts/AGENTS.md`](../prompts/AGENTS.md) copies the complete first English
scope-limit block from [HERO Anti-OverDefense at fae28f06b6b6f1582b1554b334dc1525b807cedc](https://github.com/wanshuiyin/HERO-Anti-OverDefense/blob/fae28f06b6b6f1582b1554b334dc1525b807cedc/RULES.md),
MIT, copyright 2026 Ruofeng Yang, and includes its original license in the same
file for standalone copying. The skill-authoring Skill conceptually applies the
evaluation-record guidance and retains its own original grant in
[`HERO-LICENSE.txt`](../skills/skill-authoring/references/HERO-LICENSE.txt).
[Adoption scope](hero-adoption.md) records source and loading boundaries. No
upstream runtime, hook or installer is bundled.

## CLI dependencies

`zt` binaries contain the Go runtime and the applicable modules below.
The `third-party/` files are distribution materials, not documentation:
their unmodified originals are embedded in `zt license` and accompany both
release archives and CI binary artifacts. A link alone does not replace them.

| Pinned module | License and upstream source | Retained original, relative to repository root |
| --- | --- | --- |
| Go runtime / standard library, minimum Go 1.26.0 | [BSD-3-Clause, The Go Authors](https://go.googlesource.com/go/+/refs/tags/go1.26.0/LICENSE) | `third-party/GO-BSD.txt` |
| `github.com/spf13/cobra@v1.10.2` | [Apache-2.0, The Cobra Authors](https://github.com/spf13/cobra/blob/v1.10.2/LICENSE.txt) | `third-party/COBRA-APACHE-2.0.txt` |
| `github.com/spf13/pflag@v1.0.9` | [BSD-3-Clause, Alex Ogier / The Go Authors](https://github.com/spf13/pflag/blob/v1.0.9/LICENSE) | `third-party/PFLAG-BSD-3-CLAUSE.txt` |
| `github.com/inconshreveable/mousetrap@v1.1.0` | [Apache-2.0, Alan Shreve](https://github.com/inconshreveable/mousetrap/blob/v1.1.0/LICENSE) | `third-party/MOUSETRAP-APACHE-2.0.txt` |
| `github.com/99designs/keyring@v1.2.2` | [MIT, 99designs](https://github.com/99designs/keyring/blob/v1.2.2/LICENSE) | `third-party/KEYRING-MIT.txt` |
| `github.com/99designs/go-keychain@v0.0.0-20191008050251-8e49817e8af4` | [MIT, Keybase](https://github.com/99designs/go-keychain/blob/8e49817e8af4/LICENSE) | `third-party/GO-KEYCHAIN-MIT.txt` |
| `github.com/danieljoos/wincred@v1.1.2` | [MIT, Daniel Joos](https://github.com/danieljoos/wincred/blob/v1.1.2/LICENSE) | `third-party/WINCRED-MIT.txt` |
| `github.com/gsterjov/go-libsecret@v0.0.0-20161001094733-a6f4afe4910c` | [MIT, Goran Sterjov](https://github.com/gsterjov/go-libsecret/blob/a6f4afe4910c/LICENSE) | `third-party/GO-LIBSECRET-MIT.txt` |
| `github.com/godbus/dbus@v0.0.0-20190726142602-4481cbc300e2` | [BSD-2-Clause, Georg Reinke](https://github.com/godbus/dbus/blob/4481cbc300e2/LICENSE) | `third-party/GODBUS-DBUS-BSD-2-CLAUSE.txt` |
| `github.com/dvsekhvalnov/jose2go@v1.5.0` | [MIT, upstream copyright 2014](https://github.com/dvsekhvalnov/jose2go/blob/v1.5.0/LICENSE) | `third-party/JOSE2GO-MIT.txt` |
| `github.com/mtibben/percent@v0.2.1` | [MIT, Michael Tibben](https://github.com/mtibben/percent/blob/v0.2.1/LICENSE) | `third-party/PERCENT-MIT.txt` |
| `github.com/cyphar/filepath-securejoin@v0.6.1` | [BSD-3-Clause for compiled files, LICENSE.BSD](https://github.com/cyphar/filepath-securejoin/blob/v0.6.1/LICENSE.BSD); [upstream aggregate BSD-3-Clause AND MPL-2.0, COPYING.md](https://github.com/cyphar/filepath-securejoin/blob/v0.6.1/COPYING.md) | `third-party/FILEPATH-SECUREJOIN-BSD-3-CLAUSE.txt`, `third-party/FILEPATH-SECUREJOIN-COPYING.md` |
| `github.com/go-git/gcfg@v1.5.1-0.20230307220236-3a3c6141e376` | [BSD-3-Clause, Péter Surányi / The Go Authors](https://github.com/go-git/gcfg/blob/3a3c6141e376c06bf6ca92f0450e974250b4367a/LICENSE) | `third-party/GCFG-BSD-3-CLAUSE.txt` |
| `github.com/go-git/go-billy/v5@v5.9.2` | [Apache-2.0, go-billy contributors](https://github.com/go-git/go-billy/blob/v5.9.2/LICENSE) | `third-party/GO-BILLY-APACHE-2.0.txt` |
| `github.com/go-git/go-git/v5@v5.19.3` | [Apache-2.0, go-git contributors](https://github.com/go-git/go-git/blob/v5.19.3/LICENSE) | `third-party/GO-GIT-APACHE-2.0.txt` |
| `github.com/golang/groupcache@v0.0.0-20241129210726-2c02b8208cf8` | [Apache-2.0, Google](https://github.com/golang/groupcache/blob/2c02b8208cf8c02a3e358cb1d9b60950647543fc/LICENSE) | `third-party/GROUPCACHE-APACHE-2.0.txt` |
| `github.com/jbenet/go-context@v0.0.0-20150711004518-d14ea06fba99` | [MIT, Juan Batiz-Benet](https://github.com/jbenet/go-context/blob/d14ea06fba99/LICENSE) | `third-party/GO-CONTEXT-MIT.txt` |
| `github.com/pjbgf/sha1cd@v0.7.0` | [Apache-2.0, pjbgf](https://github.com/pjbgf/sha1cd/blob/v0.7.0/LICENSE) | `third-party/SHA1CD-APACHE-2.0.txt` |
| `golang.org/x/net@v0.57.0` | [BSD-3-Clause](https://go.googlesource.com/net/+/refs/tags/v0.57.0/LICENSE) and [patent grant](https://go.googlesource.com/net/+/refs/tags/v0.57.0/PATENTS), The Go Authors | `third-party/GO-BSD.txt`, `third-party/X-SYS-PATENTS.txt` |
| `golang.org/x/sys@v0.47.0` | [BSD-3-Clause](https://go.googlesource.com/sys/+/refs/tags/v0.47.0/LICENSE) and [patent grant](https://go.googlesource.com/sys/+/refs/tags/v0.47.0/PATENTS), The Go Authors | `third-party/GO-BSD.txt`, `third-party/X-SYS-PATENTS.txt` |
| `golang.org/x/term@v0.45.0` | [BSD-3-Clause](https://go.googlesource.com/term/+/refs/tags/v0.45.0/LICENSE) and [patent grant](https://go.googlesource.com/term/+/refs/tags/v0.45.0/PATENTS), The Go Authors | `third-party/GO-BSD.txt`, `third-party/X-SYS-PATENTS.txt` |
| `gopkg.in/warnings.v0@v0.1.2` | [BSD-2-Clause, Péter Surányi](https://github.com/go-warnings/warnings/blob/v0.1.2/LICENSE) | `third-party/WARNINGS-BSD-2-CLAUSE.txt` |

The Go 1.26.0 runtime `LICENSE` and the pinned `x/net`, `x/sys`, and `x/term`
`LICENSE` files are byte-identical; `GO-BSD.txt` retains that shared original.
Their `PATENTS` files are also byte-identical and share `X-SYS-PATENTS.txt`.
The added Apache-2.0 module originals differ byte-for-byte, so each is retained
separately with its own upstream attribution text.

`filepath-securejoin@v0.6.1` declares the project-wide SPDX expression
`BSD-3-Clause AND MPL-2.0`, but its upstream `COPYING.md` assigns licenses per
file and says file headers take precedence. For the selected go-git import set,
`go list -deps -json` on darwin/amd64, linux/amd64, and windows/amd64 selects only
`filepath-securejoin/doc.go`, `join.go`, `vfs.go`, and
`internal/consts/consts.go` from that module; each has a BSD-3-Clause header.
No `pathrs-lite` package appears in those target selections. Both the unmodified
upstream `LICENSE.BSD` and `COPYING.md` are retained; `COPYING.md` includes the
full MPL-2.0 text as context for the module's other files. This records
file-level attribution and package selection, not legal clearance.

Platform-specific modules stay in this index because the CLI ships for Linux,
macOS and Windows. Keyring compiles additional backends even when zt does not
select them; `go-libsecret` is pure Go over D-Bus, not the C LGPL library.
Cobra's documentation-generator modules are not compiled into `zt`.

The existing license tests compare `go list -deps` for the host CLI with this
reviewed table and check that each mapped original exists. They neither author
the table nor decide whether its legal attribution is correct. A new linked
module or version without a reviewed row fails; human review must supply it.
Other-platform additions still require review on their native platform.

## UI source and separately installed dependencies

Keep the original MIT notices for copied shadcn composition and the Radix / Vaul
patches, plus the react-remove-scroll-bar supplement missing from its npm
archive. These are the four files in `packages/ui/third-party/`.

| Material | Terms / source |
| --- | --- |
| shadcn/ui composition | [MIT, shadcn](https://github.com/shadcn-ui/ui/blob/main/LICENSE.md) |
| Radix patch context | [MIT, WorkOS](https://github.com/radix-ui/primitives/blob/main/LICENSE) |
| Vaul patch context | [MIT, Emil Kowalski](https://github.com/emilkowalski/vaul/blob/v1.1.2/LICENSE.md) |
| react-remove-scroll-bar supplement | [MIT, Anton Korzunov](https://github.com/theKashey/react-remove-scroll-bar/blob/7301c160fda44cb8cf2b9fdfde61efad35736196/LICENSE) |
| Noto Sans / Serif families | [OFL-1.1, Noto Project Authors](https://github.com/notofonts/latin-greek-cyrillic) |
| Noto CJK variants | [OFL-1.1, Adobe; Reserved Font Name Source](https://github.com/notofonts/noto-cjk) |
| Noto Color Emoji | [OFL-1.1, Google](https://github.com/googlefonts/noto-emoji/blob/main/fonts/LICENSE) |

Font binaries are fetched from Google Fonts, not bundled or vendored here.
Keep the font declaration and links, not duplicate OFL texts for unshipped
fonts. A consumer that starts self-hosting must retain the notices and comply
with the licenses of the font binaries it actually distributes.

Private preview caches and browser failure traces can contain font binaries.
The four manually retained originals in `packages/ui/test/font-notices/`
cover those families without repeated identical copies. CI carries them with
the evidence, not in the npm package or registry item; manual evidence exports
must do the same. Keep the original traces, rather than stripping font responses.

Ordinary npm dependencies remain separate upstream packages, not MIT code
owned by tools. In particular, eslint-plugin-sonarjs declares LGPL-3.0-only,
axe-core declares MPL-2.0, and lucide-react declares ISC. Consumers must review
their installed versions and redistribution obligations. The private UI test
fixtures bundle dependencies; their existing bundler license records retain
actual notices with CI evidence. Those records do not generate this document.

## Maintenance and verification

When changing dependencies or distribution:

1. Review the actual code/assets shipped and upstream terms at the pinned version.
2. Update this index manually; retain required originals, reusing byte-identical
   notices only when their provenance is explicit. Do not rewrite copyright lines.
3. Run `node --test scripts/*.test.mjs`, `go test ./...`, `go vet ./...` and
   the affected package's native pack/source checks. These verify delivery,
   not legal clearance or license compatibility.

Published artifacts and existing consumer copies are not rewritten by this
cleanup. Publication, replacement of dependencies and consumer migration remain
separate owner decisions.
