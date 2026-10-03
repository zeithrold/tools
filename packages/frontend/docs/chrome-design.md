# Chrome design contract

The `@ztd-me/ui` source registry delivers compact chrome and menu motion for the generic React shell.
Its public symbols, preferences, persistence and consumer ownership remain compatible with the 0.2 API.
Production consumer migrations require fresh public source verification and consumer-owned acceptance.

## Visual decisions

- Appbar controls use 13px text, a 16px appearance icon and a 14px locale chevron. They have no boxed
  border at rest. Hover and open states use a 32px visual surface within the retained 44px hit area.
- The 2px accent focus outline remains visible around the full hit area. Compact appearance text still
  hides below 640px; the locale label stays visible. Consumer navigation remains a consumer slot.
- `--ztd-border` is a decorative divider/outline token, with subdued light and dark values for all six
  palettes. It must not be the sole signal for a control, focus, selection or error. Foreground, muted
  text, accent and focus colors remain unchanged. Consumer surfaces that use this token also soften;
  consumer-authored hardcoded borders require a separate consumer change.
- Appearance content retains its 220px width. Locale content uses its trigger width with a 128px minimum.
- Menus use a subtle semantic shadow. Entry fades and scales from 95%, with a 4px side-aware translation
  over 150ms; exit reverses over 100ms. Reduced motion removes these animations. Coarse-pointer choices
  have a 44px minimum height.
- UI action/navigation/status icons use Lucide components, rather than Unicode or emoji substitutes.
  Decorative SVGs are hidden from assistive technology; icon-only controls retain accessible names.
  Sizes and strokes follow semantic roles. Genuine prose, mathematics, user content and intentional
  Noto Color Emoji are preserved. This rule also applies to consumer-owned UI during migration.

The motion follows the CSS state, side and transform-origin approach in the official
[shadcn DropdownMenu](https://ui.shadcn.com/docs/components/radix/dropdown-menu) and
[Select](https://ui.shadcn.com/docs/components/radix/select), with scoped plain CSS instead of Tailwind
animation utilities. [Radix Presence](https://www.radix-ui.com/primitives/docs/guides/animation) retains
content during CSS exit animation. No animation dependency is needed.

Radix owns these runtime custom properties, explicitly inventoried by the CSS gate:
`--radix-dropdown-menu-content-transform-origin`, `--radix-select-content-transform-origin`, and
`--radix-select-trigger-width`. The inventory does not exempt other undefined variables.

## Verification and evidence

The browser suite checks compact controls, retained hit areas, entry/exit motion, keyboard selection,
focus restoration, reduced motion, inert cleanup, coarse-pointer targets and narrow reflow. It also
covers all palettes, light/dark/system, SSR hydration, fullscreen, storage rejection and Axe.
Consumers must verify their own business surfaces and obtain visual/interaction acceptance.

The shared decorative border token affects consumer surfaces that use it; hardcoded consumer borders
require their own reviewed change. Font metrics vary with the requested Noto family and glyphs.
Follow the [typography contract](fonts.md) and verify actual rendered glyphs and native weights.

Source installation checks the selected Radix declarations with strict library checking. Preserve the
[declaration-only compatibility patch](radix-compatibility.md); it does not alter interaction runtime.
When dependencies change, inspect the consumer's resolved React/Radix graph and keep provider/context/
hooks on the same compatible source revision.

`scripts/capture-review.mjs` can capture a built synthetic consumer for a PR. It writes screenshots,
video and measurements to `.artifacts/visual-<label>/`; local font previews require an explicit preview
label. Browser gates retain reports and captures under `.artifacts/` for CI upload. Keep those disposable
outputs outside source documentation. See [verification](verification.md) for native commands, public
installation acceptance and the distinction between real Google delivery and local preview evidence.
