# Noto typography through Google Fonts

The source UI uses the Google Fonts CSS2 API directly. `styles/fonts.css` imports Noto Sans and
Noto Sans SC/JP/KR at weights 400, 500, 600 and 700, plus **Noto Color Emoji** at its native weight 400.
The owner selected color for all emoji. There is no monochrome-first fallback, bundled font binary or
Fontsource dependency. The emoji token and ordinary Latin/CJK stacks use the same color family.

An earlier monochrome candidate caused Chromium to use its platform color font for the VS16 heart
(`❤️`) and rainbow flag (`🏳️‍🌈`). An isolated Chromium 151 preview tested official Google font bytes
downloaded with normal TLS: Noto Color Emoji renders all six sample sequences as single custom-font
glyphs. The production query now selects that family. The complete default and post-merge public gates
must verify actual remote browser delivery; local preview evidence does not replace them.

Noto Sans handles English and Latin text. Simplified Chinese uses Noto Sans SC. Elements marked
`lang="ja"` or `lang="ko"` prioritize the matching CJK family for regional glyph forms. Set the
correct language on content and portals; changing the shared UI locale does not translate business
content. The shared controls still support English and Simplified Chinese, independently of the font
coverage. Consumers adding other languages should verify the appropriate Noto variant and its license.

Noto Serif and its appropriate CJK variants are permitted for content that benefits from serif
typography. The ordinary UI does not load unused Serif families. Add an explicit Google Fonts API
request and semantic consumer token when introducing serif content; avoid a blanket body replacement.

`--ztd-font-sans` and `--ztd-font-emoji` are the delivered font tokens. Latin and CJK precede emoji in the
ordinary text stack, preserving normal digits and text symbols. Use `.ztd-emoji` for an explicitly composed emoji
sequence. Meaningful emoji need a text equivalent or accessible label; decorative emoji are hidden
from assistive technology. `font-synthesis: none` prevents fabricated weights. Symbols and icons are
not substituted with platform emoji as part of this design.

## Network, performance and CSP

The browser requests CSS from `https://fonts.googleapis.com` and font subsets from
`https://fonts.gstatic.com`. This is a third-party network dependency: those services receive normal
request information, including the client IP address and request headers. Account for these requests
in the consumer's own privacy policy and deployment review. See the
[Google Fonts privacy FAQ](https://developers.google.com/fonts/faq/privacy).

The API supplies browser-appropriate, Unicode-ranged subsets. The browser downloads the subsets used
by rendered glyphs and weights, rather than every declared CJK subset. The request specifies required
weights and `display=swap`; native fallback keeps text readable during loading or an outage. No local
font hosting is silently substituted. A failed request means the intended Noto rendering is unverified.
For an optional consumer optimization, preconnect to the two origins in its existing document head
(with `crossorigin` for gstatic). Do not send dynamic/private page text in a `text=` API query.
See the [CSS2 API](https://developers.google.com/fonts/docs/css2) for supported request syntax.

Merge these origins into the consumer's existing CSP, preserving its script and nonce policy:

```text
style-src-elem ... https://fonts.googleapis.com;
font-src ... https://fonts.gstatic.com;
```

If the policy uses `style-src` without `style-src-elem`, include the API origin in that directive.
The synthetic verification server also permits Radix style attributes and nonce-bearing style
elements, as required by the existing primitives. The registry does not rewrite application CSP.
Test the real production headers, network availability and browser font usage before acceptance.

The source SHA, API query and dependency lock are pinned. Google controls the returned CSS and font
files and may update them; remote font bytes are not made immutable by the source pin. Browser checks
record actual font identities and resource transfers. The specimen retains a guard of fewer than
80 font requests. Its draft 2,000,000-byte assertion currently fails on the full all-color specimen;
this provisional threshold is under review, not an accepted product page-size requirement. Measure
normal English/Chinese cold loads and warm cache reuse separately before selecting performance budgets.
The harness observes actual font requests and encoded response sizes; API URLs need not end in a font
file extension.

## Licensing and evidence

The five loaded Noto families use SIL Open Font License 1.1. Complete copyright/license notices are
delivered in `third-party/NOTO-*-OFL.txt`, including `NOTO-COLOR-EMOJI-OFL.txt`. The earlier monochrome
notice is also retained; preserve these six notices with the source. Fonts are served by Google, rather than
redistributed as registry or npm assets. The shadcn MIT notice remains separate and intact.

The browser specimen checks real rendered font usage through Chromium's
[CSS.getPlatformFontsForNode](https://chromedevtools.github.io/devtools-protocol/tot/CSS/#method-getPlatformFontsForNode),
not just `font-family` or `document.fonts.check`. It exercises English, Chinese, Japanese, Korean,
weight 600 (loaded faces, distinct Latin weight metrics and no synthesis), mixed text and emoji,
VS16, skin tone, ZWJ family/technologist/rainbow flag and a regional
flag. A composed emoji must render as one custom Noto Color Emoji glyph. Axe and captures cover the specimen.
This does not certify every Unicode sequence, browser, screen reader or network region.
