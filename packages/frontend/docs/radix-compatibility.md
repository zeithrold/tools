# Radix Select declaration compatibility

The runtime retains shadcn's new-york/Radix composition and the intended Select keyboard interaction. It uses exact individual Radix primitives: dropdown-menu 2.1.24, select 2.3.7 and slot 1.3.3. These are the same primitive versions contained in the consumers' radix-ui 1.6.7 umbrella; direct imports avoid loading unrelated primitives and do not replace Radix.

The existing Memory, Website and Showcase snapshots use TypeScript 6.0.3, @types/react 19.3.0, `strict: true`, `skipLibCheck: true`, and do not set `exactOptionalPropertyTypes`. Memory/Showcase use the same Select declaration path, but their current compiler settings do not check this conflict. The new package keeps `exactOptionalPropertyTypes: true` and `skipLibCheck: false`.

A minimal probe against that exact existing installation produced:

| exactOptionalPropertyTypes | skipLibCheck | Result |
| --- | --- | --- |
| true | false | TS2320 in react-select/dist/index.d.mts |
| false | false | Pass |
| true | true | Pass |

The error is: `SelectPopperPositionProps` cannot simultaneously extend `PopperContentProps` and `SelectPopperPrivateProps`; their `onPlaced` properties are not identical. The private declaration indexes an optional property, adding explicit `undefined` under exact optional checking.

The pinned pnpm patch changes only the corresponding inheritance line in `dist/index.d.ts` and `dist/index.d.mts`:

```ts
// Original
interface SelectPopperPositionProps extends PopperContentProps, SelectPopperPrivateProps {}
// Patched: private keys have one owner, while all other public properties remain.
interface SelectPopperPositionProps extends
  Omit<PopperContentProps, keyof SelectPopperPrivateProps>, SelectPopperPrivateProps {}
```

No JavaScript, exported public Select props, casts, compiler flags, security policies or upstream licenses change. `pnpm-workspace.yaml` and the lockfile pin the patch hash. The regression test reconstructs the original declaration and requires TS2320, then requires the patched dependency to compile with strict library checking.

The published shell's public declarations use explicit React/own types and do not expose Select's internal declarations. An independent packed consumer uses normal, unpatched registry dependencies and full declaration checking, and verifies actual Select behavior, focus/inert restoration and keyboard navigation.

Remove the patch when a stable Select release has the corrected inheritance and passes the same regression, source build and unpatched packed-consumer checks. Update the exact dependency and patch lock together; do not suppress the failing declaration or switch controls to conceal it.
