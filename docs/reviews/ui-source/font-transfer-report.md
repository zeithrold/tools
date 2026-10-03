# Bounded Google Fonts transfer diagnosis

This report uses actual Chromium Google Fonts requests in
[CI run 37103043378](https://github.com/zeithrold/tools/actions/runs/37103043378), source
`217e92afa01a0626019677cc0b97976c48fe23c2`, Node 22. Node 24 repeats the same scenarios.
It measures isolated synthetic pages, not website/showcase/memory production traffic.
The [complete resource evidence](font-transfer-google-ci-node22.json) includes each URL and family.

Each cold page starts in a separate fresh browser context. Its warm phase is an immediate ordinary
reload in that same context. All cold font/CSS responses report cache misses; all warm ones report
cache hits and zero received HTTP response bytes. DNS, TLS setup and rendering time are not measured.
Results are per page and phase, not cumulative totals across languages or the test suite.

## Response bytes and scenarios

MB means 1,000,000 bytes; MiB means 1,048,576 bytes. The draft test uses a 2,000,000-byte cap on font
responses alone. Chromium CDP reports encoded HTTP response bytes, including headers and encoded
payload but excluding socket/TLS framing. Decoded body sizes refer to HTTP decompression: WOFF2 font
containers remain compressed. Warm decoded sizes are references to the earlier cold response body,
not bytes transmitted again. The table excludes application HTML, JS, local CSS and images.

| Scenario | Cold font files | Cold font response bytes | Cold API CSS response bytes | Combined cold MB | Warm HTTP bytes |
| --- | ---: | ---: | ---: | ---: | ---: |
| English shell, normal navigation/content, closed menus | 1 | 36,371 | 341,420 | 0.377791 | 0 |
| Chinese chrome and synthetic Chinese content, closed menus | 8 | 431,162 | 342,522 | 0.773684 | 0 |
| Full English/Chinese/Japanese/Korean + bold/mixed/emoji specimen | 25 | 2,694,230 | 342,522 | 3.036752 | 0 |

The Chinese scenario is a short synthetic page with its own heading, paragraph and card; it does not
represent every Chinese page or glyph inventory. It includes the shared shell's English identity and
Chinese controls. The full specimen intentionally covers multiple languages and weights on one page.

| Family/resource | English cold files / HTTP bytes | Chinese cold files / HTTP bytes | Full specimen cold files / HTTP bytes | Full decoded body bytes |
| --- | ---: | ---: | ---: | ---: |
| Noto Sans | 1 / 36,371 | 1 / 24,411 | 1 / 24,165 | 24,004 |
| Noto Sans SC | 0 / 0 | 7 / 406,751 | 7 / 413,175 | 412,432 |
| Noto Sans JP | 0 / 0 | 0 / 0 | 4 / 133,682 | 133,008 |
| Noto Sans KR | 0 / 0 | 0 / 0 | 3 / 48,012 | 47,684 |
| Noto Color Emoji | 0 / 0 | 0 / 0 | 10 / 2,075,196 | 2,073,208 |
| Google API CSS | 1 / 341,420 | 1 / 342,522 | 1 / 342,522 | 1,372,529 |

The API returned different font/CSS bodies between fresh requests; its bytes are mutable despite the
pinned source query. Do not generalize these exact sizes into immutable font-version guarantees.
All six intentional emoji sequences remain in the specimen and render as single custom color glyphs.
Color Emoji accounts for about 77% of the full specimen's font-response bytes. Its broad composed
sequences touch ten API Unicode-range subsets; a normal page does not eagerly download them all.

No Noto Serif or monochrome Noto Emoji family is declared or requested. Japanese and Korean files are
also absent from the normal English/Chinese pages. There are no repeated resource URLs within a cold
phase. The current API repeats font-face declarations for four weights, but the corresponding variable
font URL is downloaded once. The full specimen deliberately overlaps CJK coverage when exercising
language-specific text and its mixed Chinese-tagged bold line; this is not normal appbar usage.

## Corrections and feasible reductions

- Four Unicode icon substitutes in the review fixture became Lucide SVGs: About's external arrow,
  the intro star, Explore's arrow and the card arrow. The shared exported controls already used Lucide.
  The local comparison removes about 143 KB of font responses from the English page, because those
  symbols had pulled in SC/emoji subsets. Genuine emoji samples, prose and mathematical text stay intact.
- A normal-TLS, read-only API comparison of `wght@400;500;600;700` with `wght@400..700` returns exactly
  the same 368 font URLs. It reduces font-face declarations from 1,439 to 368 and decoded CSS from
  1,304,025 to 333,750 bytes. Local gzip estimates fall from 349,876 to 89,691 bytes; those are estimates,
  not a browser measurement of the alternative. See [comparison evidence](font-variable-range-comparison.json).
  This is a promising visual-preserving optimization; adopting it still requires the full remote
  glyph/weight/CSP suite, with loaded weight-range coverage checked correctly. The query is unchanged.
- Preserve Unicode-range demand loading and Google caching. Do not preload every family/subset or
  request unused Serif fonts. Preconnect may reduce connection latency but does not reduce these bytes.
- Do not remove emoji cases, subset only the test's text, substitute platform fonts, self-host binaries,
  or weaken glyph/weight/CSP checks to meet a byte threshold.

## Proposed measurement policy, awaiting review

Do not raise one full-specimen cap merely to obtain a green run. Measure normal English and Chinese
initial pages separately, with font files and API CSS separately accounted for, and test warm cache
reuse separately. The combined observations are approximately 0.378 MB and 0.774 MB before any
variable-range optimization. A future representative fixture budget could be 0.5 MB / 1.0 MB combined
cold responses, plus a 10 KB warm response allowance, with an independent CSS ceiling. These are
proposals grounded in these fixtures, not accepted product requirements or universal page budgets.
Recalibrate after the visual-preserving CSS optimization and consumer-owned representative pages.

Keep the broad specimen as the full rendering/weight/sequence/CSP matrix and retain its transfer report
and request-count guard; its size should not be presented as normal appbar cost. The draft 2 MB assertion
currently remains unchanged and fails. At this source SHA, each Node version passes 23 of 24 browser
cases; only that assertion fails at 2,693,326 / 2,693,398 font-response bytes in its independent specimen.
Strict lint/CSS/types/build, eleven units, ESLint/helper-package gates and Go pass. The PR remains draft;
budget handling, final-head checks, owner merge and fresh public full-SHA installation are still required.
