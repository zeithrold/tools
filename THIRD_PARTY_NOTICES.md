# Third-party notices

First-party work is MIT, copyright 2026 Zeithrold; see `LICENSE`.
Dependencies retain their own copyrights and licenses.

The manually maintained component declarations, pinned upstream links and
distribution boundaries are in
[docs/licensing.md](https://github.com/zeithrold/tools/blob/main/docs/licensing.md).

CLI distributions retain the Go runtime's BSD notice and the applicable
Apache-2.0, BSD, BSD-2-Clause and MIT module originals in `third-party/`.
Those files are also embedded in the executable and readable with `zt license`;
the root MIT grant does not replace them. The Go 1.26.0 runtime and pinned
x/net, x/sys and x/term license texts are byte-identical and share
`GO-BSD.txt`; their identical patent grants share `X-SYS-PATENTS.txt`.

`filepath-securejoin` has a mixed, file-level BSD-3-Clause/MPL-2.0 upstream
license. The selected platform file lists include only its BSD-3-Clause files;
both the original `LICENSE.BSD` and upstream `COPYING.md` are retained. The
index documents the exact scope without asserting legal clearance.

UI deliveries retain required copied-code and patch notices. Remote Google
Fonts are not bundled; their OFL terms and upstream links are documented in
the index. Separately installed npm dependencies retain their own terms.

`prompts/AGENTS.md` copies the complete English HERO Anti-OverDefense scope-limit
block, copyright 2026 Ruofeng Yang, under MIT, and includes its original grant.
The skill-authoring Skill conceptually applies its evaluation-record guidance
and retains `skills/skill-authoring/references/HERO-LICENSE.txt` for independent
distribution. Source, scope and loading boundaries are in `docs/hero-adoption.md`.
