# Web interaction and layout

Read the project's design contract and the ui-design principles first. Prefer
native semantic controls and explicit labels. Check keyboard operation, focus
visibility and order, error association, and live feedback. Verify reflow and
content preservation at narrow widths and enlarged text. Test the actual
supported browsers and assistive technology paths when the change requires them.

Use the framework's native testing and lint tools. Automated accessibility checks
are useful evidence, but they do not replace checking rendered content,
interaction states and recovery in the browser. Keep framework APIs and project
component choices in local guidance.

Follow [browser review](web-browser-review.md) for route and state, translated
labels, keyboard and Axe coverage, built-Worker evidence and retained captures.
Use `frontend-engineering` for deterministic hydration and `verification` for
executable gates.

For styled scrollbars, provider nonce and portal forwarding, and modal or sheet
composition, implementation details belong to the optional
`frontend-engineering` Skill or the project's component documentation.
Review the result here without requiring a sibling installation. A styled scroll container must render an actual bounded
height and scroll range while native wheel, touch and keyboard scrolling still
work. Modal and sheet content needs a title, a description and an explicitly
localized reachable close control. Verify focus trapping and restoration, Escape,
nested overlays, native scrolling and narrow reflow in the real browser.
