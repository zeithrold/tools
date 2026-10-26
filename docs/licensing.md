# Licensing and distribution contract

All first-party code and documentation in this checkout use MIT with the existing
`Copyright (c) 2026 Zeithrold` notice. The root `LICENSE` is the canonical full text. SPDX package and
Skill metadata use `MIT`. Preserve complete licenses with copies or substantial portions, including
consumer-owned adaptations. Third-party terms are listed separately in
[the upstream licensing boundary](../THIRD_PARTY_NOTICES.md).

| First-party component | License coverage | Distribution checks |
| --- | --- | --- |
| Root Go module, `cmd/zt`, `internal/*`, examples, scripts, workflows and documentation | Root `LICENSE`; CLI embeds MIT, the original Go BSD notice and every linked Go module's own license text | `go test ./...`, `go vet ./...`, `zt license`; CLI archive coverage test |
| `@ztd-me/eslint` | `packages/eslint-config/LICENSE`, `license: MIT`, explicit npm `files` inclusion | `pnpm run test:pack` checks packed license bytes and metadata before fresh consumer verification |
| `@ztd-me/frontend-checks` | `packages/frontend-checks/LICENSE`, `license: MIT`, explicit npm `files` inclusion | `pnpm run test:pack` checks packed license bytes and metadata before fresh consumer verification |
| `@ztd-me/ui` source/components/styles and private harness | `packages/ui/LICENSE`, `license: MIT`; original shadcn, Radix, scroll-bar dependency and Noto notices preserved | Private `test:pack` checks MIT and all nine upstream notices; UI harness remains private |
| Root registry inventory and generated UI payload | `meta.license: MIT`; full `LICENSE` delivered beside `THIRD_PARTY_NOTICES.md` and nine upstream notices | Registry coverage test compares every delivered file to canonical source; source-install smoke checks copied notices |
| Every bundled Skill under `skills/`; see the [catalog](global-skills.md#bundled-skills) | Each complete directory includes `LICENSE` and `license: MIT` frontmatter | Embedded Skill sync test verifies licenses in every installed directory and repeat-sync idempotence |
| UI consumer templates and reusable references | Root MIT grant; distributed registry README includes the boundary; Skill references travel with their directory license | Registry and Skill delivery checks |
| Private compiled UI test fixtures uploaded as CI evidence | Fixture MIT license and original UI notices; Vite emits full actual bundled-dependency license texts | Packed and source consumer builds assert nonempty complete dependency license records |
| CLI release archives and CI binary artifact | Full MIT text, Go BSD notice, third-party boundary and every linked Go module's original license text beside the binary | Release uses `scripts/package-cli.sh`; tar/zip coverage test exercises all six targets |

The `zt` binary is not a single-license artifact. Root MIT covers first-party code and the Go runtime
is BSD-3-Clause. The linked libraries keep their own terms: `99designs/keyring` and its pure-Go
helpers are MIT, cobra and mousetrap are Apache-2.0, pflag, `golang.org/x/sys` and
`golang.org/x/term` are BSD-3-Clause, and `godbus/dbus` (the module path keyring requires, without
`/v5`) is BSD-2-Clause. keyring compiles every backend into one package per platform, so its file,
prompt and Linux D-Bus code is in the binary even though `zt` allows only the OS backends; the notice
set is therefore wider than the three selected backends, and it lists nothing that is not compiled.
`go-libsecret` is pure Go over D-Bus, so no C `libsecret` or other LGPL library is linked by that
name. The complete unmodified texts live in `third-party/` and are embedded through `LicenseFiles`,
so `zt license` reproduces them from any distribution of the binary and does not depend on this
checkout. The boundary, pinned versions and per-module purpose are described in
[the upstream notice](../THIRD_PARTY_NOTICES.md); the root MIT grant does not convert those terms.

Run the repository-level coverage checks with:

```sh
node --test scripts/*.test.mjs
go test ./...
go vet ./...
```

Run the native package checks in each package after a frozen pnpm install. The source payload is
generated with the pinned command in [contributor guidance](../CONTRIBUTING.md). Local packing and
source installation validate candidate content. An approved full SHA and fresh public installation
remain required before consumer acceptance; local checks cannot establish that public gate.

Previously published npm artifacts and existing consumer copies are not rewritten by this change.
Their archived license metadata and upstream notices remain as shipped. Future publication and
consumer migration are separate owner actions; this contract does not authorize either.
