# Chrome refinement candidate

This local candidate refines the 0.2.0 shell and delivers it as the `@ztd-me/ui` source registry item.
Its public symbols, preferences, persistence, consumer ownership and Radix versions stay compatible.
It has not been published or migrated into a production consumer.

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

The motion follows the CSS state, side and transform-origin approach in the official
[shadcn DropdownMenu](https://ui.shadcn.com/docs/components/radix/dropdown-menu) and
[Select](https://ui.shadcn.com/docs/components/radix/select), with scoped plain CSS instead of Tailwind
animation utilities. [Radix Presence](https://www.radix-ui.com/primitives/docs/guides/animation) retains
content during CSS exit animation. No animation dependency is needed.

Radix owns these runtime custom properties, explicitly inventoried by the CSS gate:
`--radix-dropdown-menu-content-transform-origin`, `--radix-select-content-transform-origin`, and
`--radix-select-trigger-width`. The inventory does not exempt other undefined variables.

## Evidence and limits

The attached Ocean/light Chinese screenshot was inspected directly. The Library transfer failed twice
with `library file transfer failed: download failed`; no local original screenshot bytes were available.
The review fixture is an English synthetic consumer with its own navigation, content and footer, not a
reproduction or modification of a production project. The approximately matching content viewport is
1500×860; additional captures use 390×844 and 1920×1080.

`scripts/capture-review.mjs` captures the unchanged package and candidate using the same built consumer
fixture. Its measured 0.2.0 Chinese controls are 80×44 and 106×44, with 14px text. Its menu has computed
`animation-name: none` and zero animations. This reproduces the reported absent motion independently of
the dependency duplication hypothesis.

The refined Chinese controls measure 66×44 and 90×44 with 13px text and transparent appbar borders.
Mobile appearance remains 44×44. The content uses the quieter decorative border token; this fixture
does not establish how any consumer's hardcoded borders will render after migration.

`scripts/audit-dependencies.mjs` audits the freshly installed published 0.2.0 consumer. The package is
compiled with TypeScript, not bundled: the artifact retains external Radix imports. Root and radio item
resolve the same DropdownMenu module; components resolve the consumer's React; provider and hook resolve
one preferences context and its object identity matches. The isolated dependency graph has one installed
version of each Radix package. These findings do not prove that production consumers cannot contain
incompatible versions or duplicate peer graphs. No production dependency audit was performed here.

Source installation directly imports Radix declarations and therefore exposes the upstream Select
2.3.7/Popper 1.3.7 TS2320 that the old compiled public API hides from a consumer. The registry includes
the already tested declaration-only patch and an explicit pnpm configuration recipe. Strict library
checking stays enabled. This declaration defect is separate from missing motion or duplicate instances.

Browser checks cover entry/exit, focus restoration, keyboard selection, reduced motion, inert cleanup,
coarse-pointer hit areas and reflow. Existing checks cover all palettes, light/dark/system, SSR hydration,
fullscreen, storage rejection and Axe. Automated success does not replace owner visual review.
