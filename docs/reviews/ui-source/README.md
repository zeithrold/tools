# UI source review evidence

These are native Chromium captures of an isolated synthetic consumer, not a production deployment.
The owner approved the compact chrome/source direction; merge and final consumer acceptance remain
with the owner. The before image uses the published frontend 0.2.0. Current captures use the same
fixture and the owner-authorized **local Noto font preview** in Cloud. Production registry CSS uses
the Google Fonts API directly; these images do not verify browser API/network/CSP delivery.

The local preview downloads the intended official Google Noto font files with normal TLS and serves
them from ignored test artifacts. Font binaries, preview adapters and download scripts are absent
from the source registry. Cloud Chromium reports `ERR_CERT_AUTHORITY_INVALID` for the real API,
while normal CLI TLS returns HTTP 200. No certificate store or TLS security setting was changed.

| State | Evidence |
| --- | --- |
| Published 0.2.0 / Ocean light | [Before](before-0.2.0.png) |
| Noto preview / Ocean light, Chinese controls, 1500×860 | [Desktop](noto-local-preview-desktop.png) |
| Noto preview / Ocean light, Chinese controls, 390×844 | [Mobile](noto-local-preview-mobile.png) |
| Noto preview / appearance popup | [Menu](noto-local-preview-menu.png) |
| Noto preview / Neutral dark | [Dark](noto-local-preview-dark.png) |
| Noto preview / appearance and locale entry/exit | [Motion clip](noto-local-preview-motion.mp4) |
| Approved Noto Color Emoji / multilingual specimen, 1100×850 | [Color-font preview](noto-color-local-preview-fonts.png) |

Current Chinese controls are 66×44 and 90×44 with 13px text; the mobile appearance control is 44×44.
Borders are transparent at rest, with a 32px hover/open surface inside the 44px hit target. Measurements
and detected CSS animations are recorded in [the capture receipt](noto-local-preview-measurements.json).

English/CJK font usage, loaded weight 600, distinct Latin weight metrics and no synthesis are tested
through actual Chromium rendering. The owner selected Noto Color Emoji for all emoji. The API query,
tokens, ordinary text stacks and sequence tests now select that family; no monochrome-first fallback
is used. The earlier monochrome candidate failed the VS16 heart and rainbow flag checks because
Chromium selected a platform color font.

The [isolated alternative probe](noto-emoji-alternatives-local-preview.json) records all six sequences
for three font stacks. Noto Color Emoji alone renders six custom color glyphs. Noto Emoji followed by
Noto Color Emoji renders four custom monochrome glyphs and the two affected sequences in Google's
custom color font. These isolated probes informed the owner's all-color choice; they do not establish
actual browser API/network/CSP delivery.
The [downloaded subset metadata](noto-emoji-variation-tables.json) shows base heart/white-flag mappings
in the monochrome font but no FE0F variation mappings for them; the color subsets have those mappings.
That is consistent with Chromium's observed VS16 fallback; it does not establish universal browser
behavior or a missing base glyph.

The [approved-color capture receipt](noto-color-local-preview-fonts.json) verifies six single custom
Noto Color Emoji glyphs in the fresh 42-file source consumer, with no browser errors. That capture
uses the authorized local font preview; actual remote verification is recorded by CI separately.

Default GitHub CI uses the actual Google API. At source `dc9e01b18606f84ff26ea20df47458926fcfd816`,
Node 22 and 24 each passed 22 of 23 browser scenarios, including remote English/CJK fonts, actual weight 600,
bounded font transfers and the CSP regression before/after Axe; only the emoji sequence check fails.
See [run 37100048735](https://github.com/zeithrold/tools/actions/runs/37100048735). That run preceded the
approved all-color implementation; the PR checks must pass on its final source SHA.
After owner merge, fresh public source installation with the full approved SHA must pass before
consumer migration acceptance. These captures do not replace either gate.

![Noto local preview, Ocean/light desktop](noto-local-preview-desktop.png)

![Noto local preview, mobile](noto-local-preview-mobile.png)
