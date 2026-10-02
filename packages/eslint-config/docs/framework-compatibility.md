# App Router export compatibility

Version 0.1.0 reports `react-refresh/only-export-components` on an otherwise valid vinext server layout exporting `metadata` beside its default component. This was reproduced from the public npm installation with ESLint 10.11.0 and TypeScript 6.0.3; the TSX file was included in the strict project and produced a real rule error rather than an ignored-file diagnostic.

Antfu 9.5.1 detects the package named `next`, but not `vinext`. Its Next export allowances apply to all React files, and installed Vite enables primitive constant exports. Version 0.1.1 makes the ordinary React export rule deterministic and introduces explicit `react.framework: 'next' | 'vinext'`, with an optional literal relative `appDir`. The selected research rules and numerical limits remain unchanged.

## Framework contract

[Next.js metadata documentation](https://nextjs.org/docs/app/api-reference/functions/generate-metadata) restricts metadata exports to server page/layout modules. [Route segment configuration](https://nextjs.org/docs/app/api-reference/file-conventions/route-segment-config) supplies the other recognized route exports. Older dynamic/revalidation/cache options remain framework-mode dependent; the Next.js compiler owns those restrictions. The current Next.js documentation also defines the `instant` and `prefetch` route exports; they are specific to the Next profile.

[Next.js Fast Refresh documentation](https://nextjs.org/docs/architecture/fast-refresh) explains the behavior of ordinary modules exporting both components and other values. The [upstream React Refresh rule](https://github.com/ArnaudBarre/eslint-plugin-react-refresh) offers framework export allowances, but a global allowance does not prove that metadata belongs in arbitrary components or client modules.

Vinext was inspected at upstream commit `e5ada16269f18317aad9c9070d9ff64f62699e88`, package version 1.0.1. Its [metadata implementation](https://github.com/cloudflare/vinext/blob/e5ada16269f18317aad9c9070d9ff64f62699e88/packages/vinext/src/shims/metadata.tsx) handles layout/page metadata in server rendering. Its [development route handling](https://github.com/cloudflare/vinext/blob/e5ada16269f18317aad9c9070d9ff64f62699e88/packages/vinext/src/server/dev-route-files.ts) recognizes app page/layout files and excludes private directories. Vinext's segment configuration and prefetch policy do not establish support for Next's per-segment `prefetch` export.

## Validation

Package tests exercise both framework profiles with real typed TSX fixtures; root and `src/app` paths; route groups, dynamic segments and parallel slots; explicit monorepo directories; React file-scope intersections; plain `.js` routes; and client directives parsed by both JS and TypeScript parsers. Negative cases retain errors for ordinary components, private/non-route files, client metadata/constants, arbitrary helper exports and unsupported framework names/scopes. The packed consumer checks the new API declarations and server/ordinary/client behavior independently of repository imports.

An isolated project installed the actual published vinext 1.0.1, Vite 8.3.2, `@vitejs/plugin-react` 6.1.1, `@vitejs/plugin-rsc` 0.5.35 and React 19.2.6 using pnpm 11.19.0. A server layout's metadata appeared in the SSR title, and editing the metadata changed the rendered title without restarting the dev server. This verifies server metadata handling and development invalidation; it does not claim a browser client-state preservation test or a complete Next.js/vinext application build.

The wrapper delegates component/export analysis to the pinned React Refresh rule 0.5.7. It recognizes a `'use client'` directive before selecting any server allowances and scans the selected `.js`/`.ts` route modules even when they contain no JSX. Framework builds and TypeScript remain required to validate export values, metadata exclusivity and supported configuration combinations.
