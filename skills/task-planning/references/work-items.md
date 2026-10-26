# Work items and acceptance

A work item is a unit of change someone can finish, hand over and have verified
without reading the rest of the plan. That is the whole test of its granularity:
"the backend" is not an item, and one item per edited file is not a plan.

## Each item carries five facts

```
Changes:   what behavior or artifact becomes different
Consumes:  files, symbols, data, events, decisions it needs (with paths)
Produces:  what later items or callers depend on, named precisely
Done when: the observable state that means it is finished
Checked by: the command, flow or review that demonstrates it, with expected output
```

Guidance per field:

- **Consumes** must already exist or be produced by an earlier item. An item that
  needs a decision nobody has made is blocked, not ready; do not list it as next.
- **Produces** is an interface promise. Vague produces ("the API changes") means
  the next item guesses, and the guess becomes the interface.
- **Done when** is observable from outside the item: a response field, a stored
  row, a page behavior, a migration result. "Code written" is not done.
- **Checked by** names the exact command or flow and the expected result, because
  a check whose expected output is unstated cannot fail — only surprise. Where the
  claim is runtime behavior, the check exercises the real entry point; syntax or
  type checks support a narrower claim.

A defect fix states both halves where feasible: the previous revision fails the
new check and the changed revision passes it. Without that, the check proves the
suite is green, not that the defect is fixed.

## Ordering

Order by dependency, then by how early a wrong assumption gets caught. Put the
step that could invalidate the plan — the interface spike, the migration on a
populated schema, the measurement — as early as its inputs allow, so a rewrite
lands on a paragraph instead of on merged code.

An item may start when its inputs exist and no earlier item owns the same
interface. Everything else waiting is waiting on a decision, not on capacity, and
adding people does not shorten that.

## Worked example (synthetic)

Requirement: the audit log endpoint must page, so large accounts can load history
without scanning everything.

Scope decision recorded first: cursor pagination, not offset. Deciding
constraint: rows are inserted continuously, and offsets skip or repeat entries
under concurrent inserts, which is a correctness problem in an audit view, not a
quality-of-life one. This is the option that meets the constraint, so no menu of
alternatives is carried forward.

```
1. Cursor read path in the audit store
   Changes:   audit reads return a page plus an opaque next cursor
   Consumes:  internal/audit/store.go (List), schema of audit_events
   Produces:  ListPage(ctx, cursor, limit) (Page, error); Page{Events, NextCursor}
   Done when: paging 250 seeded rows yields 3 pages, 250 unique ids, and an
              insert between pages neither duplicates nor skips a row
   Checked by: go test ./internal/audit -run TestListPage
   Expected:  ok; page sizes 100/100/50; ids unique across pages
   Depends on: nothing

2. Endpoint accepts limit and cursor
   Changes:   GET /audit gains limit (default 50, max 200) and cursor; adds
              next_cursor to the response
   Consumes:  item 1; internal/api/audit.go; docs/openapi.yaml
   Produces:  request parameters and next_cursor field, consumed by the web
              history view
   Done when: a request without parameters returns the first page as before;
              an invalid cursor returns 400 with a code the client can branch on
   Checked by: go test ./internal/api -run TestAuditPagination, then the local
              binary with curl against seeded data
   Expected:  ok; default page 50; 400 + {"code":"invalid_cursor"}
   Depends on: item 1

3. Web history view appends pages
   Changes:   the audit view loads more instead of replacing its list
   Consumes:  item 2's parameters and response shape
   Produces:  nothing other items need
   Done when: scrolling loads a second page and no row appears twice
   Checked by: pnpm exec playwright test audit-history.spec.ts
   Expected:  pass; 100 rows after two loads; ids unique
   Depends on: item 2
```

What this example demonstrates:

- Item 2 is blocked on item 1 even though its code is trivial; the interface is
  the dependency.
- The expected output is written before the check runs, so a check that passes
  without paging (or without asserting uniqueness) is visibly not acceptance.
- Nothing here estimates time: an estimate neither makes the order correct nor
  survives the document it was written in.
- Item 3 verifies a UI behavior no backend test can establish, and item 2 verifies
  compatibility no UI test would catch.

## When evidence changes the plan

If item 1's check shows the cursor cannot be stable without a new index, that is
not a reason to add a fallback branch: the item gains the index, and the plan
records the resulting one-way door — creating the index online is safe, removing
it later needs a maintenance window. If the check shows cursor paging cannot meet
the requirement at all, the plan returns to the scope decision instead of growing
special cases — the same trigger described by the complexity reset.

## What does not belong in items

- A restatement of the eventual diff, function by function.
- Ceremony items: "write tests", "update docs", "get review" stand as acceptance
  checks and handoffs, not as phases with no observable result.
- Time estimates and confidence numbers that no later evidence updates.
- Items whose only consumer is the plan itself.

## Handing items to other owners

When an item goes to another agent or person, it needs the exclusive files it may
edit, the interface it must not change without asking, the permission it runs
with, and the exact check plus expected output. The plan supplies the content; the
agent-team Skill supplies the coordination contract when multiple owners share a
revision, and a single-owner plan needs no team at all.
