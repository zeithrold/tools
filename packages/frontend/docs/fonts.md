# Noto typography through Google Fonts

The source UI uses the Google Fonts CSS2 API directly. `styles/fonts.css` imports Noto Sans and
Noto Sans SC/JP/KR at weights 400, 500, 600 and 700, plus **Noto Emoji** at 400 and 600. Noto Emoji is
the monochrome family; this candidate does not select the separate Noto Color Emoji family. Confirm
that visual choice during owner review. There are no bundled font binaries or Fontsource dependencies.

The current monochrome candidate has an acceptance blocker: Chromium renders the VS16 heart with its
platform color font despite the requested Noto Emoji family. Plain emoji, skin-tone technologist,
family and flag use the custom font. The sequence test deliberately fails on platform fallback; owner
selection of Google Noto Color Emoji or an explicit Noto Color Emoji fallback is pending. Do not treat
the current mono candidate as fully verified or substitute a system font silently.

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
record actual font identities and resource transfers. Their sample budget is fewer than 80 WOFF2
requests and under 2 MB transferred for the multilingual specimen, not a universal page-size guarantee.

## Licensing and evidence

The five Noto families use SIL Open Font License 1.1. Complete copyright/license notices are delivered
in `third-party/NOTO-*-OFL.txt`; preserve them with the source. Fonts are served by Google, rather than
redistributed as registry or npm assets. The shadcn MIT notice remains separate and intact.

The browser specimen checks real rendered font usage through Chromium's
[CSS.getPlatformFontsForNode](https://chromedevtools.github.io/devtools-protocol/tot/CSS/#method-getPlatformFontsForNode),
not just `font-family` or `document.fonts.check`. It exercises English, Chinese, Japanese, Korean,
weight 600 (loaded faces, distinct Latin weight metrics and no synthesis), mixed text and emoji,
VS16, skin tone, ZWJ family/technologist/rainbow flag and a regional
flag. A composed emoji must render as one custom Noto Emoji glyph. Axe and captures cover the specimen.
This does not certify every Unicode sequence, browser, screen reader or network region.
