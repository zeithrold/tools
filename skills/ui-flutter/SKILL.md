---
name: ui-flutter
description: Apply shared UI principles to Flutter widgets, text scaling, focus, semantics, and native-device verification.
---

# Flutter UI review

Read the project's design contract and `ui-foundation` first. Use its chosen component foundation and theme tokens. Verify text scaling, safe areas, keyboard insets, focus, semantics, and reachable actions across the changed screens and sheets. Keep loading, empty, error, and recovery states meaningful without fabricated product data.

Use widget tests for deterministic layout and semantics contracts. Use a real device or simulator for native behavior that widget tests cannot establish. Record tested devices, locales, themes, and text scales; distinguish captures from visual approval.
