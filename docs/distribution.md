# Distribution and installation

This page is the contract for zt release assets, the GitHub release pipeline and
installation through [mise](https://mise.jdx.dev/). It does not authorize a
release: creating a `v*` tag and publishing the GitHub release remain explicit
owner actions.

**Current status:** no release exists yet, no `v*` tag has been pushed for this
contract, and the workflow has not been executed for a release. The native build
and credential gates below are defined but are not evidence of a passing native
run until each target job actually executes. `examples/mise.toml` therefore uses
an example version.

## Release assets

Every release publishes exactly six archives and one checksum file:

```text
zt_vX.Y.Z_darwin_amd64.tar.gz
zt_vX.Y.Z_darwin_arm64.tar.gz
zt_vX.Y.Z_linux_amd64.tar.gz
zt_vX.Y.Z_linux_arm64.tar.gz
zt_vX.Y.Z_windows_amd64.zip
zt_vX.Y.Z_windows_arm64.zip
SHA256SUMS
```

Each archive contains `zt` (`zt.exe` on Windows), `LICENSE`,
`THIRD_PARTY_NOTICES.md` and **every** notice under `third-party/` at the same
relative paths. `scripts/package-cli.sh` copies the whole directory, so a new Go
dependency cannot ship without its license text. `SHA256SUMS` lists all six
assets in byte order.

Release binaries are built from the tagged commit with
`-X github.com/zeithrold/tools.releaseVersion=<tag>`, so `zt version` prints the
tag. They also embed the public GitHub App client ID through
`-X main.githubClientID=<client id>`; an empty value means device flow is not
configured. The client ID is a public identity, not a secret: the release reads
only the `ZEITHROLD_GITHUB_CLIENT_ID` secret, accepts `Iv1.<hex>` or `Iv23…`
GitHub App client IDs, rejects numeric app IDs and OAuth `Ov23…` values, and
never prints a rejected value. No private key or token is embedded.

## Release pipeline gates

`.github/workflows/release.yml` runs on an existing `v*` tag and never creates
tags or merges. A release only publishes when all of these pass:

| Gate | What must actually run |
| --- | --- |
| Source | Tag format, `go mod download` + `go mod verify` under `-mod=readonly`, `go test ./...`, `go vet ./...`, repository scripts tests |
| Native build (per target) | Build on the target's own OS runner with `CGO_ENABLED=1` on Darwin and `0` on Linux/Windows; run the built binary natively and record `target`, `version`, `native_run=passed` |
| Package | Reject a target without a native-run receipt; assemble the six archives; verify names, checksums, exact archive layout, license bytes and the host-native archive run |
| Native credential store | `ZT_KEYSTORE_INTEGRATION=1 go test ./internal/keystore -run '^TestOSStoreIntegration$'` on every native runner; a skipped test fails the gate |
| Publish | Re-verify `SHA256SUMS`, create GitHub build provenance attestations, `gh release create --verify-tag` with minimal permissions |

The credential gate is env-gated: without `ZT_KEYSTORE_INTEGRATION=1` the test
skips, and the workflow treats a skip as a failure. On macOS the gate creates a
throwaway keychain, unlocks it and passes it through `ZT_KEYSTORE_KEYCHAIN`; on
Linux the runner starts an isolated D-Bus session with GNOME Keyring and seeds
an unlocked `login` Secret Service collection with `secret-tool` before the
test; on Windows the test uses the runner's Credential Manager. The adapters use
the real backends (Darwin Keychain, Linux Secret Service over GoDBus, Windows
WinCred), not fakes. The Linux adapter targets the GNOME-standard `login`
collection, so a KWallet-only image needs an equivalent default collection or
would fail the gate. If the integration test is missing from the tagged tree the
gate fails. Failed build directories and packaging output are uploaded as
artifacts.

## Installing with mise

mise's `github` backend installs the release asset and verifies its checksum, so
no custom plugin or pipe-to-shell installer is needed:

```toml
# examples/mise.toml
[tools]
"github:zeithrold/tools" = { version = "0.1.0", matching = "zt_" }
```

```sh
mise install
mise exec -- zt version
mise use -g 'github:zeithrold/tools[matching=zt_]@0.1.0'   # global install
```

`matching = "zt_"` excludes `SHA256SUMS` and keeps the six zt archives; mise
selects the archive for the current OS/architecture from the file names. Do not
disable checksum or attestation verification (`github_attestations = false`) to
work around a failure. A local `mise.lock`, if you commit one, records the URL
and checksum for each platform.

`mise upgrade` moves to a newer version and pinning an older version in the same
config rolls back; mise owns install directories, shims and removal
(`mise uninstall github:zeithrold/tools@<version>` removes one version).
Upgrading does **not** run `zt sync`: Skills under `~/.agents/skills` change only
when you run `zt sync` explicitly, and unmanaged personal Skills are preserved.

## Replacing an existing installation

Find every copy first; PATH order decides which one runs:

```sh
command -v zt
type -a zt          # Windows: where zt
mise ls
```

The previously used install in this environment is a mise `go:` backend:

```sh
mise unuse -g --no-prune 'go:github.com/zeithrold/tools/cmd/zt@main'
mise uninstall 'go:github.com/zeithrold/tools/cmd/zt@main'
```

`unuse` removes the global request and `uninstall` removes the installed copy.
Keep that order: if the request stays in global config, a later `mise install`
reinstalls the old build. `unuse -g` without `--no-prune` also removes the
installed version it no longer needs; `--no-prune` lets you verify before
removing. Do not delete mise shims by hand.

A binary produced by `go install` (for example `$(go env GOBIN)/zt` or
`$(go env GOPATH)/bin/zt`) is **outside mise**: `mise uninstall` cannot remove
it, and no zt command deletes it. Inspect provenance before touching the file:

```sh
go version -m "$(command -v zt)"
ls -l "$(command -v zt)"
```

Check the module path, version, `vcs.revision` and build settings, confirm the
exact path, and remove only that file with explicit user approval, or leave it
and make sure mise's shim wins on PATH. Deleting a file is not a secure erase.

## Verifying a release

```sh
node scripts/verify-release.mjs --dist dist --tag vX.Y.Z
node scripts/verify-mise-release.mjs --build   # local fake release, no publish
```

The first checks the six names, `SHA256SUMS`, archive layout and license bytes,
and runs the host archive's `zt version`/`zt license`. The second serves a local
fixture, lets mise install into isolated data directories, asserts that only the
host archive is selected, checks the installed executable and license files, and
proves a corrupted archive is rejected against `SHA256SUMS`. `--build` compiles
only the host archive and writes clearly labelled placeholder archives for the
other five targets, because the fixture verifies mise's host selection and
checksum handling, not cross-platform builds; pass `--dist` to verify archives
produced by CI instead. The fixture is a loopback HTTP API because mise's macOS
build verifies TLS through Security.framework and ignores `SSL_CERT_FILE`;
trusting a local test CA would modify the user's keychain, so the fixture does
not do that. It never creates a tag, release, or real installation, and it is
not a substitute for a real release run.

## Authorization

A release requires the owner to push the tag and let the workflow publish. The
workflow does not create tags, merge pull requests, or install anything on a
developer machine.
