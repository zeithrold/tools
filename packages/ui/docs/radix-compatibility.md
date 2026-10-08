# Radix Select declaration compatibility

The runtime retains shadcn's new-york/Radix composition and the intended Select keyboard interaction. Its exact individual primitives are inventoried below; the manifest, registry and lock retain these pins. Direct imports avoid loading unrelated primitives and do not replace Radix. Consumers audit their own resolved versions before composing primitives from different packages.

Compiler options affect whether the upstream declaration conflict is checked. The private UI source harness keeps `strict: true`, `exactOptionalPropertyTypes: true` and `skipLibCheck: false`; consumer compiler settings and dependency versions remain consumer-owned.

The pinned Select declaration regression covers these compiler settings:

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

The public primitive exports now reach Radix Select declarations. Both the independent packed consumer
and source consumer install this same documented pnpm patch and retain full declaration checking.
The patch is included with the source/private artifact; preserve it when consuming the enhanced Select
API under these strict compiler settings. Native Select remains available for standard select/option forms.

Remove the patch when a stable Select release has the corrected inheritance and passes the same regression, source build and unpatched packed-consumer checks. Update the exact dependency and patch lock together; do not suppress the failing declaration or switch controls to conceal it.

## Primitive dependency inventory

| @radix-ui/react- package | Version |
| --- | --- |
| accordion | 1.2.20 |
| alert-dialog | 1.1.23 |
| checkbox | 1.3.11 |
| collapsible | 1.1.20 |
| dialog | 1.1.23 |
| dropdown-menu | 2.1.24 |
| scroll-area | 1.2.18 |
| select | 2.3.7 (declaration-only patch) |
| slot | 1.3.3 |
| tabs | 1.1.21 |
| tooltip | 1.2.16 |

All retain upstream MIT licensing; preserve WorkOS's full RADIX-MIT.txt notice and installed dependency
notices. The selection respects the existing release-age/trust policy; no new exception is required.
The private identity audit compares the resolved React, preference context and shared Radix modules in
its actual disposable consumer. Its result does not certify an uninspected production dependency graph.
