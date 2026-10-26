# Flutter review

Read the project's design contract and the ui-design principles first. Use the
project's chosen component foundation, packages and theme tokens as they are; the
web reference's React, Radix and CSS details do not apply here. Verify text
scaling, safe areas, keyboard insets, focus, semantics and reachable actions
across the changed screens and sheets. Keep loading, empty, error and recovery
states meaningful without fabricated product data.

Use widget tests for deterministic layout and semantics contracts. Widget tests
run in a test harness, not on a platform runtime, so use a real device or a
native simulator/emulator for platform behavior they cannot establish: OS input,
text scaling and locale on the platform, insets, lifecycle, accessibility
services and navigation. A simulator carries the platform runtime but not the
hardware, so treat real-device-only characteristics such as performance, cameras,
sensors and device keyboards as a separate claim and label them unverified when
no device is available. Record the tested devices or simulators, locales, themes
and text scales, and distinguish captures from visual approval.

The Noto family, emoji and delivery requirements in the principle sections remain
in force; merging platform guidance does not authorize a new font-policy
exception. If a native platform cannot satisfy the approved route, report the
constraint and obtain an explicit project decision rather than silently
vendoring fonts or changing families. Verify rendered glyphs on the platform.
