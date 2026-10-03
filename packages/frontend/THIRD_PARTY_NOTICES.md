# Third-party notices

The package's self-owned code is MIT licensed. This does not relicense dependencies or font assets.

- UI composition follows shadcn/ui's new-york Radix foundation (MIT, copyright shadcn). Its license is included in `dist/assets/SHADCN-MIT.txt`.
- Radix primitives and Lucide React remain separately installed dependencies under their upstream licenses. Preserve their distributed copyright and license notices.
- The current source harness loads Noto Sans, Noto Sans SC/JP/KR and Noto Color Emoji directly through
  the Google Fonts API. Their complete SIL Open Font License 1.1 notices ship in
  `dist/assets/NOTO-*-OFL.txt` and the source registry. The earlier monochrome notice is retained too,
  for six complete font notices. No font binaries are bundled. This source change
  does not alter the already published 0.2.0 artifact or its Inter license notices.

Sources: [shadcn/ui](https://github.com/shadcn-ui/ui), [Google Fonts](https://github.com/google/fonts),
[Noto CJK](https://github.com/notofonts/noto-cjk), and [Noto Emoji](https://github.com/googlefonts/noto-emoji).
