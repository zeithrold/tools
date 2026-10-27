# Third-party notices

The package's self-owned code is MIT licensed. This does not relicense dependencies or font assets.

- UI composition follows shadcn/ui's new-york Radix foundation (MIT, copyright shadcn). Its license is included in `dist/assets/SHADCN-MIT.txt`.
- Radix primitives (MIT, copyright 2022 WorkOS; preserved in `dist/assets/RADIX-MIT.txt`) and Lucide React remain separately installed dependencies under their upstream licenses. Preserve their distributed copyright and license notices.
- The source registry's Select 2.3.7 declaration patch includes Radix source context (MIT, copyright
  2022 WorkOS). Its original notice ships in `dist/assets/RADIX-MIT.txt` and the source item.
- `react-remove-scroll-bar@2.3.8`, an installed Radix dependency, declares MIT but omits the full
  notice from its npm archive. The original upstream Anton Korzunov notice is retained in
  `dist/assets/REACT-REMOVE-SCROLL-BAR-MIT.txt`, source delivery and bundled fixture license records.
  Source: [upstream LICENSE](https://github.com/theKashey/react-remove-scroll-bar/blob/7301c160fda44cb8cf2b9fdfde61efad35736196/LICENSE).
- The current source harness loads Noto Sans, Noto Sans SC/JP/KR and Noto Color Emoji directly through
  the Google Fonts API. Those font families are SIL Open Font License 1.1 software served by Google:
  no font binary, vendored font package or OFL copy is bundled, and `dist/assets/` carries none. Their
  upstream notices stay at [Google Fonts `ofl`](https://github.com/google/fonts/tree/main/ofl), for
  example [Noto Sans](https://github.com/google/fonts/blob/main/ofl/notosans/OFL.txt),
  [Noto Sans SC](https://github.com/google/fonts/blob/main/ofl/notosanssc/OFL.txt) and
  [Noto Color Emoji](https://github.com/google/fonts/blob/main/ofl/notocoloremoji/OFL.txt). The
  repository [licensing boundary](https://github.com/zeithrold/tools/blob/main/docs/licensing.md)
  records which third-party material must ship a notice. This source change does not alter the already
  published 0.2.0 artifact.

Sources: [shadcn/ui](https://github.com/shadcn-ui/ui) and
[Google Fonts](https://github.com/google/fonts).

## Tailwind component foundation

Tailwind CSS and tailwind-merge are MIT licensed. The source item requires Tailwind CSS 4.3.3
compilation and pins tailwind-merge 3.7.0. Vaul 1.1.2 (MIT, Emil Kowalski) supplies Drawer gestures;
VAUL-MIT.txt retains its notice. The included exact patch removes runtime stylesheet injection only;
reviewed static behavioral CSS retains the native presence, drag and snap-point selectors under
components layer. It fixes the upstream invalid hitarea selector, removes a duplicate selector and
uses semantic handle color and scoped keyframe names. No CSP policy is weakened.

Noto Serif, Noto Serif SC, Noto Serif JP and Noto Serif KR are opt-in Google Fonts families under the
same upstream SIL OFL 1.1 terms, delivered as request-only CSS rather than as bundled font software.
UI defaults remain Noto Sans with CJK and Color Emoji.
