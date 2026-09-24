# Ledger tooling assessment queue

This is an inventory of the current local Ledger checkouts, not a migration plan or a claim that their checks passed. Keep `ledger-tooling`, the backend, and the Flutter app operational while evaluating one capability at a time.

| Existing capability | First assessment | Current action |
| --- | --- | --- |
| Cross-platform argument, path, timeout, and process handling | Potentially general CLI infrastructure | Compare behavior with another stack before extracting |
| `governance.json` command arrays and `just` recipes | Useful native entry-point evidence; 723 backend and 510 app config lines today | Read them through `inspect`/`plan`; do not copy the schema wholesale |
| Go test, five fuzz targets, and accounting mutation command | Real project commands with distinct cost and prerequisites | Discover and plan existing recipes; leave execution and gates in Ledger |
| Exact-copy `skills-check` | Protects immutable copies but conflicts with project customization | Use managed Skill ownership and conflict detection for new shared Skills; leave existing Ledger copies alone |
| Debug evidence workflow | Process ideas may generalize | Extract only after checking another repository's debugging needs |
| Coverage thresholds, policy ratchet, review gate | Valuable but reflect Ledger's chosen governance | Do not impose global thresholds or review policy |
| Currency and API contract generation | Ledger domain and integration behavior | Keep in Ledger |
| Flutter UI report and `ledger-ui` Skill | Contains reusable review principles and Ledger-specific components | Extract principles into `ui-foundation`; retain design tokens, Material selection, and transaction details in Ledger |

For each later extraction, record: source behavior, a second consumer, intended shared contract, project-owned override path, tests, and a reversible rollout. No existing Ledger command should be removed merely because `zt` can discover it.
