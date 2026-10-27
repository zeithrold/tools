# Font notices for run evidence only

Original SIL Open Font License 1.1 texts for the Noto families this UI requests from the Google Fonts
API. They are kept **only** to accompany run evidence that can contain font software: a retained
Playwright failure trace stores the `*.woff2` responses the browser fetched, and the explicit local
preview caches the font files it downloads. The UI source item and npm package distribute no font
software and ship none of these files; their four MIT texts remain in `packages/ui/third-party/`.

| File | Covers |
| --- | --- |
| `NOTO-SANS-OFL.txt` | Noto Sans and Noto Serif (identical original bytes) |
| `NOTO-SANS-SC-OFL.txt` | Noto Sans SC, JP and KR (identical original bytes) |
| `NOTO-SERIF-SC-OFL.txt` | Noto Serif SC, JP and KR (identical original bytes) |
| `NOTO-COLOR-EMOJI-OFL.txt` | Noto Color Emoji |

The monochrome `NOTO-EMOJI-OFL.txt` is deliberately absent: the delivered CSS never requests that
family. One file stands for a group only because its members were byte-identical in this repository's
history; every byte here is an unmodified original copy. Maintain this set by hand.

CI copies this directory unchanged to `.artifacts/font-notices/` and uploads it beside the
browser evidence, whose traces stay as recorded rather than having font responses scrubbed. A manual
local-preview export must carry the same four notices.
