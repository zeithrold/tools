# Provenance for adopted mechanisms

Provenance answers where each rule came from and what it became. It applies to
conceptual adoption as much as to copied text: adopting a mechanism still means
the source's assumptions, license and failure history travel with it. Examples
are synthetic.

## Record fields

| Field | Content |
| --- | --- |
| Source | Repository, document or internal synthesis; revision and file or heading |
| Terms | License, notice obligations, redistribution limits |
| Adoption kind | Verbatim text, paraphrase, concept/mechanism, observed behavior |
| Destination | Exact file and section that now carries it |
| Change | What was kept, removed and rewritten for this repository |
| Validation | Static review, behavior trial, or `unknown` when unvalidated |
| Reviewer | Who accepted the adoption |

## Conceptual adoption still needs a row

Rewriting a mechanism in original words changes the expression, not the origin.
A destination row lets a later maintainer find the upstream revision, check the
license again, see which assumptions were imported, and revise both sides
consistently. "Original text, no source" is usually incomplete rather than true.

Synthetic example: a guide adopts the idea of deriving expected values outside
the implementation. The source is an internal research synthesis section; license
review clears the concept and the paraphrased wording; the destination is a
reference file; the change removes product-specific examples and replaces them
with synthetic ones; validation is static review only, recorded as such.

## Templates are not the problem

- Do not copy an unlicensed or unknown-license template wholesale; paraphrasing
  is not a rights workaround.
- Generic templates with compatible terms may be adapted when they carry no
  consumer's production domain, branding, credentials, deployment target or
  business rule.
- Operational examples, matrices, worked cases and long procedures belong in
  `references/`, not in the main body, even when they are useful.
- A synthetic example written for this repository is original material; say so
  when it resembles a source, and keep the source row anyway.

## Destination discipline

A mechanism is adopted once per concern. Do not restate the same rule in the
main body, a reference and a project instruction with drifting wording; name the
owner and let other files reference it. When a destination is removed later, its
provenance row is removed or marked retired so no reader follows a dead source.

## Validation and notices

Record only checks that actually ran. Frontmatter, links and license lookups are
static checks; they do not show that the guidance changes behavior. Preserve
required license text and notices beside the distributed directory, and never
put a secret, personal path or private source into provenance or examples.

## HERO evaluation records

The exact-content/revision guidance in `evaluation-design.md` (“Freeze before
the first trial” and its examples) and `task-evaluation.md` (“Freeze a cohort”)
conceptually applies the hash and rubric limits from
[HERO `RULES.md` at fae28f06b6b6f1582b1554b334dc1525b807cedc](https://github.com/wanshuiyin/HERO-Anti-OverDefense/blob/fae28f06b6b6f1582b1554b334dc1525b807cedc/RULES.md).
It replaces mandatory new hashes with retained exact content or existing native
revision records, while preserving fixed inputs and change visibility. It does
not remove existing native records or establish behavioral effectiveness.
The owner selected this application; independent behavioral acceptance remains
unverified. Upstream is MIT, copyright 2026 Ruofeng Yang; the complete original
grant is retained in [HERO-LICENSE.txt](HERO-LICENSE.txt).
