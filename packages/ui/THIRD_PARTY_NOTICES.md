# Third-party notices

The package's self-owned code is MIT licensed. This does not relicense dependencies or font assets.

- UI composition follows shadcn/ui's new-york Radix foundation (MIT, copyright shadcn). Its license is included in `dist/assets/SHADCN-MIT.txt`.
- Radix primitives (MIT, copyright 2022 WorkOS; preserved in `dist/assets/RADIX-MIT.txt`) and Lucide React remain separately installed dependencies under their upstream licenses. Preserve their distributed copyright and license notices.
- The current source harness loads Noto Sans, Noto Sans SC/JP/KR and Noto Color Emoji directly through
  the Google Fonts API. Their complete SIL Open Font License 1.1 notices ship in
  `dist/assets/NOTO-*-OFL.txt` and the source registry. The earlier monochrome notice is retained too,
  for six complete font notices. No font binaries are bundled. This source change
  does not alter the already published 0.2.0 artifact or its Inter license notices.

Sources: [shadcn/ui](https://github.com/shadcn-ui/ui), [Google Fonts](https://github.com/google/fonts),
[Noto CJK](https://github.com/notofonts/noto-cjk), and [Noto Emoji](https://github.com/googlefonts/noto-emoji).

## Tailwind component foundation

Tailwind CSS and tailwind-merge are MIT licensed. The source item requires Tailwind CSS 4.3.3
compilation and pins tailwind-merge 3.7.0. Vaul 1.1.2 (MIT, Emil Kowalski) supplies Drawer gestures;
VAUL-MIT.txt retains its notice. The included exact patch removes runtime stylesheet injection only;
reviewed static behavioral CSS retains the native presence, drag and snap-point selectors under
components layer. It fixes the upstream invalid hitarea selector, removes a duplicate selector and
uses semantic handle color and scoped keyframe names. No CSP policy is weakened.

Noto Serif, Noto Serif SC, Noto Serif JP and Noto Serif KR are opt-in Google Fonts families, each with
its upstream SIL OFL 1.1 notice retained. UI defaults remain Noto Sans with CJK and Color Emoji.
