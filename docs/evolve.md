# Offline evolution candidates

`zt evolve prepare` and `zt evolve plan` implement the local candidate stage.
They do not apply changes, evaluate behavior, authenticate, fetch Git objects,
run checks or create PRs. `check`, `submit`, `status` and activation are not
implemented. The [proposal Skill](../skills/evolution-proposals/SKILL.md) explains
the wider method; its future lifecycle is not a shipped command contract.

## Why structured replacements

The input is explicit complete file replacements, not an executable patch stream
or arbitrary unified diff. This keeps the first slice bounded and binds every
existing file to its exact preimage. It intentionally excludes deletes, file
modes, scripts, binaries and changes outside one allowed Skill.

## Separate policy from proposal

An operator supplies the trusted policy explicitly. Do not obtain it from a
candidate or silently inherit it from a consumer's `zt.json`.

Illustrative policy; replace the commit placeholder with the full lowercase
commit ID present in the selected local upstream:

```json
{
  "schemaVersion": 1,
  "host": "github.example",
  "repository": "example/skills",
  "baseCommit": "FULL_LOWERCASE_BASE_COMMIT",
  "allowedSkills": ["example-skill"],
  "requiredChecks": ["structure", "behavior"]
}
```

`requiredChecks` are descriptive identifiers, not executable commands or evidence
that those checks exist. Unknown fields, including command definitions, are
rejected. Target identity, allowed scope and check names come from this policy,
not the proposal.

A proposal declares one Skill and one origin, then provides file replacements:

```json
{
  "schemaVersion": 1,
  "skill": "example-skill",
  "provenance": {
    "repository": "example/source",
    "commit": "FULL_LOWERCASE_SOURCE_COMMIT",
    "license": "MIT",
    "reason": "Document a previously missing failure and recovery path"
  },
  "files": [{
    "path": "skills/example-skill/references/recovery.md",
    "preimageSha256": "",
    "content": "# Recovery\n\nOriginal, reviewed guidance goes here.\n"
  }]
}
```

An empty preimage is allowed only for a new path. Existing files require their
full lowercase SHA-256 from the pinned base tree, not the working tree. Source
commit and license are declarations: the source repository is not fetched and
redistribution rights are not verified. SPDX expressions are not supported in
this slice; the license field accepts a single identifier.

Optional `evidence` entries contain only `label` and a SHA-256 `digest`; they are
opaque declarations, never dereferenced local paths or URLs.

Field names are case-sensitive and required fields must be present with their
declared types. Duplicate/unknown keys, `null`, malformed UTF-8 and unpaired
Unicode surrogate escapes are rejected, not silently repaired. An explicitly
empty content string is distinct from omitted or null content.

## Prepare and inspect

```sh
zt evolve prepare --proposal proposal.json --policy policy.json \
  --upstream /path/to/local/upstream --root /path/to/workspace --json

zt evolve plan /path/to/workspace/.zt/artifacts/evolve/CONTENT_ID/candidate.json \
  --policy policy.json --upstream /path/to/local/upstream --json
```

The selected workspace must exist. Prepare writes only its owned
`.zt/artifacts/evolve/<content-id>/candidate.json` tree, after validating the
proposal and pinned Git base. It does not modify the upstream worktree, index,
branch or consumer source. Store operator policy and proposals deliberately;
candidate content may be private and is not approved for upload merely because
it was prepared locally.

The content ID is SHA-256 of the exact canonical candidate bytes. It binds the
normalized policy, target/base, source declarations, check names, evidence
references, replacements and preimages. No timestamp or mutable workflow status
is part of that identity. Identical preparation is idempotent; a conflicting
existing directory is not overwritten.

The returned `candidatePath` is relative to the selected workspace. Resolve it
there, or pass its absolute path to `plan` when invoking from another directory.

Plan re-reads the canonical candidate, checks its identity and location, and
revalidates policy and base preimages. Location means the file is stored as
`<content-id>/candidate.json`, with the directory naming the SHA-256 of exactly
the bytes it holds. The directories above that one are not inspected: a candidate
copied out of the workspace that prepared it is still read, and is still refused
once its bytes no longer match that name. Plan creates no directory, lock, journal
or cache. Both commands use read-only local Git plumbing with lazy fetching
disabled; missing objects are an error, not permission to fetch.

System/global Git configuration, tracing destinations and replacement-object
redirection are suppressed. Local/common/worktree repository configuration must
still be read to open a repository; any `include` or `includeIf` is refused
before Git starts. A `.git` gitdir pointer or a `commondir` file that names a path
beginning or ending with whitespace, or that carries more than one line, is
refused instead of interpreted: Git keeps those bytes, so trimming them could
inspect one directory while the Git zt then starts opens another, and the preflight
would miss the include it promised to refuse. Repositories relying on includes, or
on such pointer layouts, are deliberately unsupported in this slice rather than
allowing either to read operator-home files. This is not a claim of zero local
Git-config or object-store IO.

## Limits and truthful results

- At most eight UTF-8 text files, 256 KiB each and 512 KiB total content.
- Only `skills/<selected-id>/SKILL.md`, `LICENSE`, and Markdown under
  `references/`. Absolute/traversal/backslash/control-character paths, case
  collisions, symlink/submodule targets and file/directory conflicts are refused.
- Policy: 64 KiB; proposal: 1 MiB; canonical candidate: 2 MiB.
- All required checks remain `not_run`, source/evidence declarations remain
  `declared_not_verified`, and publication remains `not_ready`.
- Independent license, privacy, review, authorization and verification evidence
  remain missing. There is deliberately no ready or approved state.

Changing policy invalidates the candidate's policy binding. A changed proposal
creates a different candidate; do not edit a stored candidate to preserve an old
approval. Interrupted preparations may leave identifiable `.prepare-*` stages.
Inspect them before cleanup rather than treating them as valid candidates.

The next runtime stages need independently controlled verification, explicit
publication approval and idempotent remote reconciliation. Passing this local
structural stage is not proof of improved Skills or an autonomous feedback loop.
