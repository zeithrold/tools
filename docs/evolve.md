# Reviewed Skill evolution

`zt evolve` implements the local candidate stage, an explicit trusted-local
verification stage, and an idempotent draft pull request stage:
`prepare`, `from-installed`, `plan`, `check`, `submit`, `status` and `upstream`. It never merges, publishes,
activates, force-updates a ref or reopens a closed pull request, and it never
generates guidance with a model. The [proposal Skill](../skills/evolution-proposals/SKILL.md)
explains the wider method; activation, rollback and revocation remain outside
this command surface.

| Command | What it does | Effects |
| --- | --- | --- |
| `prepare` | Validates a proposal against the trusted policy and a pinned local base, writes one immutable candidate | Writes only `.zt/artifacts/evolve/<content-id>/candidate.json` |
| `from-installed` | Derives replacements from one edited managed global Skill after matching the complete pinned Skill to its installation lock digest, then prepares a candidate | Same candidate-only writes as `prepare`; installation and lock remain untouched |
| `plan` | Re-reads and re-validates a stored candidate | None: no write, credential, network or execution |
| `check` | Runs the trusted validation contract against a complete materialised pinned base with the candidate applied | Executes native commands, but only with `--trusted-local-execution`; writes its own result, logs and workspace |
| `submit` | Requires a passed local verification and an approval naming the exact candidate, target, base, validation contract and public payload, then opens one draft pull request, returns an exact existing one, or completes a verified branch missing only its pull request | Remote writes only through the GitHub App credential; a serialised, monotone local journal records each step |
| `status` | Reports local state; `--remote` additionally reads the pull request and GitHub's checks | Local reads only unless `--remote` is passed |
| `upstream status` | Reports whether the complete pinned baseline is locally usable | No write, credential, network or validation execution |
| `upstream fetch` | Acquires exact Git objects through the stored GitHub App credential with `--allow-network`, or reuses a complete cache | Writes only a staged then verified bare cache under `.zt/artifacts/evolve/upstreams/`; no remote writes |

## Managed upstream baseline (no user checkout)

An explicit local `--upstream` is still supported. Alternatively, populate a
workspace-owned bare cache once, then omit `--upstream` from the existing stages:

```sh
# Local readiness only. Missing or damaged objects do not trigger a download.
zt evolve upstream status --policy policy.json --root /path/to/workspace --json

# Explicit network/credential operation. The policy supplies host, repo and commit.
zt evolve upstream fetch --policy policy.json --root /path/to/workspace \
  --allow-network --client-id Iv1.example --profile default --json

zt evolve from-installed --skill example-skill --policy policy.json \
  --root /path/to/workspace \
  --source-repository example/source --source-commit FULL_LOWERCASE_SOURCE_COMMIT \
  --license MIT --reason "Clarify recovery" --json

zt evolve plan CANDIDATE.json --policy policy.json --root /path/to/workspace --json
zt evolve check CANDIDATE.json --policy policy.json --root /path/to/workspace \
  --validation validation.json --trusted-local-execution --json
```

The cache key binds the host, repository and exact commit, not a mutable branch.
It contains the original commit (including its original signature, parents,
timestamps and message bytes), trees and all baseline blobs. A shallow history
boundary is allowed; a partial/blobless baseline is not. Fetching never checks out
files or changes a user's repository, worktree or index. It does not recursively
fetch submodules, expand LFS pointers, approve publication or run validation.

Downloads use an in-process `go-git` client and a separate repository-bound
GitHub App HTTPS transport. The REST endpoint's permissions are unchanged.
Only upload-pack discovery and reads are allowed; redirects, other repositories,
receive-pack and non-HTTPS production endpoints are refused. The existing OS
keystore and App credential refresh lifecycle are reused, with no PAT, environment
token, plaintext credential or credential-bearing Git subprocess fallback.
Tokens are never persisted in the bare repository's URL or configuration.

Downloads are staged independently and published only after native object
integrity/connectivity and complete baseline reads pass. Baselines retain the
verification limits below (4096 tracked entries, 4 MiB per file, 32 MiB total)
and regular-file mode restrictions. Discovery responses are capped at 1 MiB;
packfile payloads at 128 MiB, with a 1 MiB allowance for preceding protocol
headers and a five-minute fetch deadline. These are transfer and final baseline
limits, not a sandbox or a bound on decompression/index memory while the Git
library ingests objects. GitHub caching supports SHA-1 repositories;
explicit local upstreams retain SHA-256 support. There is no automatic eviction
or repair: a damaged existing cache is refused, not overwritten. Inspect the
reported owned path before manually removing it, or supply an explicit upstream.
An interrupted download is not a ready cache; normal error/cancellation paths
remove their own staging directory, while process termination can leave a
`.fetch-*` stage.

`upstream status` distinguishes `ready`, `missing` and `blocked`, includes the
next fetch command as an argument array, and reports authentication as
`not_checked` and checks as `not_run`. It does not claim the App can access the
repository or any validation passed. Existing-cache reuse also avoids credentials.
This is baseline readiness, not a general-purpose doctor or publication-readiness
report.

All subsequent local stages consume the cache without credentials, network or
cache writes. `plan` additionally accepts `--root` to select that workspace; it
does not infer a cache from an unrelated candidate directory. Local commands
never fill a missing cache automatically. A streaming Git request rejected with
401 may refresh the stored credential but refuse a non-replayable request;
rerunning the explicit fetch uses the refreshed credential without an unsafe
automatic replay. App access is not publication approval.

## Why structured replacements

The input is explicit complete file replacements, not an executable patch stream
or arbitrary unified diff. This keeps the slice bounded and binds every existing
file to its exact preimage. It intentionally excludes deletes, file modes,
scripts, binaries and changes outside one allowed Skill.

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

`requiredChecks` are names. The policy never carries a command: unknown fields,
including command definitions, are rejected. What each name *runs* is fixed by a
separate validation contract that must cover exactly these names. Target identity,
allowed scope and the required check names come from the policy, never from the
proposal.

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

## Prepare from an installed Skill

For a managed Skill edited under `~/.agents/skills`, the CLI can derive the
replacement set and preimage hashes instead of requiring a proposal JSON:

```sh
zt evolve from-installed --skill example-skill \
  --policy policy.json --upstream /path/to/local/upstream \
  --root /path/to/workspace \
  --source-repository example/source --source-commit FULL_LOWERCASE_SOURCE_COMMIT \
  --license MIT --reason "Clarify the failure and recovery path" --json
```

The operator still supplies the trusted policy and declares the proposed
material's source, license and reason. Those declarations do not become verified
provenance or redistribution rights. This convenience command does not generate
wording, evidence, validation results or approval; use `prepare` for a proposal
carrying explicit evidence references.

The current installation lock records a version and an aggregate content digest,
not a repository or immutable source commit. `from-installed` therefore requires
an explicit local repository or managed cache and a policy pin whose **entire selected Skill**
matches that recorded digest. It never resolves a version label as a Git ref or
guesses an upstream from the current CLI version. This also supports old managed
installations when the operator supplies a content-matching base. Matching proves
content correspondence, not the repository's ownership or the truth of source
declarations.

Only the selected managed Skill is read. Changed and new files become complete
replacements; unchanged files are excluded. No change, a missing management
record, mismatched base, deletion, symlink, non-regular file, executable base file,
unsupported changed path or exceeded bound refuses the operation without writing
a candidate. Reads are bounded to 4096 entries, 4 MiB per file and 32 MiB total
for each installed/base Skill snapshot; generated replacements retain the
existing eight-file, UTF-8 and content-size limits.

Local edits are the input, not an installation conflict to overwrite. The command
does not change the installed directory, its lock, the upstream worktree or index.
It does not fetch, merge or rebase. If the target base's Skill differs from the
installed baseline, use a matching pinned base or explicitly review replacements
for `prepare`; do not resync over edits to make this command work.

The output is the same immutable candidate/result as `prepare`, so continue with
`plan`, `check` and exact-payload approval and submission below. Identical inputs
produce the same candidate as an equivalent explicit proposal. Automatic
installation-source recording and multiple Skills per candidate
remain separate follow-ups. Managed upstream caching is described above.

## Proposal encoding

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
there, or pass its absolute path to later commands when invoking from another
directory.

Plan re-reads the canonical candidate, checks its identity and location, and
revalidates policy and base preimages. Location means the file is stored as
`<content-id>/candidate.json`, with the directory naming the SHA-256 of exactly
the bytes it holds. The directories above that one are not inspected: a candidate
copied out of the workspace that prepared it is still read, and is still refused
once its bytes no longer match that name. Plan creates no directory, lock, journal
or cache. All local Git use is read-only plumbing with lazy fetching disabled;
missing objects are an error, not permission to fetch.

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

## Trusted local verification

A validation contract is a separate operator file. It cannot be supplied by a
candidate, and it must name exactly the policy's `requiredChecks`:

```json
{
  "schemaVersion": 1,
  "checks": [
    {"name": "structure", "argv": ["node", "scripts/verify-structure.mjs"], "timeoutSeconds": 120},
    {"name": "behavior", "argv": ["pnpm", "test"], "timeoutSeconds": 600}
  ]
}
```

```sh
zt evolve check /path/to/workspace/.zt/artifacts/evolve/CONTENT_ID/candidate.json \
  --policy policy.json --upstream /path/to/local/upstream \
  --validation validation.json --root /path/to/workspace --trusted-local-execution --json
```

`check` materialises the **complete** pinned base commit, overlays exactly the
candidate replacements, and runs each command with the pinned base as the working
tree. A workspace holding only the replacements would make a native profile test a
repository that does not exist. Base entries must be regular files
(`100644`/`100755`): a symlink, submodule or other mode refuses the run instead of
approximating the tree.

The expected file set is computed before anything is written, files are created
exclusively at each exact spelling (`O_EXCL`), and the physical workspace is then
compared as a whole with that set: same paths, same modes, same content digests.
A host whose filesystem stores two distinct Git paths as one file (case folding,
Unicode normalisation) therefore refuses the run before executing anything instead
of verifying a tree it never materialised. That is a limit of the host
filesystem, not a claim about which paths a Git tree may contain: on a
case-sensitive volume a base holding `README.md` and `readme.md` verifies with both
files intact.

Execution is refused unless `--trusted-local-execution` is passed. That flag is
the operator's reviewed decision to run these commands, and the result says so:

> This is **not an OS sandbox** and **not credential isolation**. The commands run
> as native processes with the operator's own authority over this machine's files,
> network and OS keychain. A minimal environment, per-command timeouts, bounded
> logs and a fixed working tree reduce accidents; they do not contain a malicious
> candidate. Verification evidence is operator-local: the same user can alter it,
> so it is never an independent attestation.

zt never interprets `argv` through a shell. An operator who writes `sh -c ...`
is choosing that themselves; zt neither forbids nor emulates it.

Results are stored under the owned artifact tree, never inside a candidate:

```
.zt/artifacts/evolve/checks/<candidate-id>/<validation-digest>/
  tree/                 complete pinned base with the candidate applied
  out/                  the only directory offered for command output ($ZT_EVOLVE_OUT_DIR)
  logs/NN-<name>.log    bounded combined output per command
  result.json           observed statuses, exit codes, log size and SHA-256
```

The result records the exact candidate ID, validation digest, target, file
digests, the materialised workspace digest and the base entry-set digest. The
workspace is digested from the physical files before and after the run, so a
command that rewrites the inputs cannot yield a reusable `passed` record.

Reuse and submission apply the **same** record rule: the stored result must be
canonical, bound to exactly this candidate and contract, report every step as
passed with exit 0 in the contract's order, keep every retained log whose size and
SHA-256 still match, and still describe the workspace that exists on disk. A
missing log, an edited tree, a deleted workspace or a contradictory step record
(even with an overall `passed`) is not reusable and blocks submission. This is a
consistency check on local evidence, not protection against the same user writing
a complete forged record.

`check` prints its result and then exits non-zero unless the recorded status is
`passed`, matching `zt check`; a failed or blocked run is never reported as a
successful command.

## Approval and the public payload

The publication is a separate reviewed artifact. `submit --show-payload` prints
it without reading a credential, contacting the network, executing anything or
writing anything:

```sh
zt evolve submit CANDIDATE.json --policy policy.json --upstream /path/upstream \
  --validation validation.json --root /path/to/workspace --show-payload
```

That document is the exact publication, including every file's content:

```json
{
  "schemaVersion": 1,
  "host": "github.example", "repository": "example/skills",
  "baseCommit": "FULL_BASE_COMMIT", "baseCommittedAt": "2026-01-05T12:00:00Z",
  "branch": "evolve/example-skill-0123456789ab",
  "files": [{"path": "skills/example-skill/references/recovery.md", "content": "..."}],
  "commit": {"message": "...", "authorName": "zt-evolve", "authorEmail": "zt-evolve@users.noreply.github.com", "committedAt": "2026-01-05T12:00:00Z"},
  "pullRequest": {"title": "...", "body": "...", "draft": true}
}
```

Hashes bind bytes, so review must see the bytes. `publicPayloadDigest` is the
SHA-256 of that canonical document. The commit date comes from the **pinned base
commit's committer instant**, normalised to UTC, so the payload and its digest
exist before any approval and stay reproducible on replay. The author is the
fixed `zt-evolve` identity; zt never impersonates the reviewing user.

An approval is the human decision for exactly that publication:

```json
{
  "schemaVersion": 1,
  "candidateId": "CANDIDATE_CONTENT_ID",
  "target": {"host": "github.example", "repository": "example/skills", "baseCommit": "FULL_BASE_COMMIT"},
  "validationDigest": "VALIDATION_CONTRACT_SHA256",
  "publicPayloadDigest": "PUBLIC_PAYLOAD_SHA256",
  "reviewer": "a human identifier",
  "reviewedAt": "2026-02-01T09:30:00Z",
  "attestations": {"humanReview": "approved", "privacy": "approved", "license": "approved"}
}
```

Every field is compared with what zt derived itself. `approvalDigest` is the
SHA-256 of the normalised approval document, so the reviewer, review time and
each attestation are audited into the same binding. `reviewedAt` is an audited
declaration and does not influence any commit. The attestations are the operator's
own statements: zt does not verify license compatibility, privacy, provenance or
suitability, and reports them as `attested_by_operator_not_verified_by_zt`
anywhere a reader could confuse them with an observation.

The public payload is exactly: the target host, repository and pinned base with
its commit time, the branch name, the approved replacement file contents, the
declared reason and evidence labels with their digests, the declared source
repository/commit/license lines, and the deterministic commit and pull-request
text. zt adds no logs, transcripts, reasoning, absolute host paths, environment
values, credentials or unrelated source to it, and CI logs are never copied into
it. zt also does not scan the approved content for secrets: replacements, reason
and labels publish exactly the bytes a reviewer approved, so keeping secret or
private material out of a candidate remains the operator's responsibility.

## Submission, idempotency and recovery

```sh
zt evolve submit CANDIDATE.json --policy policy.json --upstream /path/upstream \
  --validation validation.json --approval approval.json --root /path/to/workspace \
  --client-id Iv1.example --profile default --json
```

`submit` requires, in this order: a canonical candidate that still matches the
policy and the pinned base; a validation contract covering the policy's checks; a
payload digest the approval names; and a local verification record for exactly
this candidate and contract, still consistent with the retained tree and logs. All
of that is checked against a read-only view of the workspace, so a missing record
or a malformed journal fails before any credential is read and before any
directory is created. Only then is the OS keystore opened, through the same GitHub
App device-flow credential as `zt auth`. There is no PAT, `gh`, environment-token
or plaintext fallback, no token in any child process environment and no token in
any result, journal or log.

It always reconciles by reading before it writes:

1. `GET /repos/{owner}/{repo}` must report exactly the policy's repository, and its
   `default_branch` is the pull request base. Only default-branch submission is
   supported.
2. The deterministic branch `evolve/<skill>-<candidate-id[:12]>` and its pull
   requests (any state) are read.
3. An existing branch or pull request is adopted only when it verifies exactly:
   head repository, head branch and base branch, open state, draft flag, approved
   title and body, commit message, `zt-evolve` author and committer identity and
   date, the single pinned base commit as parent, and the **complete** tree — every
   path, mode and blob compared with the pinned base overlaid with exactly the
   approved replacements. Extra, missing or rewritten unrelated files, and a
   truncated tree response, are refused.
4. An exact match is returned with zero writes, even if the base branch has moved.
   A closed or merged pull request on that branch is reported, never reopened. Any
   other object on that branch is refused; zt never force-updates a ref.
   Approved text is compared byte for byte: whitespace and line endings are part of
   what a reviewer approved, and Markdown indentation can change meaning, so any
   difference is drift rather than a match.
   The commit is verified as a raw Git object, not through the API's `message`
   field: zt rebuilds the object from the approved tree, parent, identity, date
   and message bytes and requires the remote object id to equal the id of that
   reconstruction. GitHub's REST view drops one trailing newline from a
   message, so comparing that view would either reject the approved bytes or
   accept a message zt never approved; the object id fails closed on any
   difference, including an added header zt did not write.
5. A branch that already holds exactly the approved commit but no pull request is
   completed by creating **only** the pull request - the ref is never re-created and
   never forced. This is the recovery path for a lost ref response.
6. Any submission that still needs **any** remote write is then gated on the base:
   the default branch head must still be the pinned base commit, otherwise the
   submission is refused and preparation, verification and approval must be
   repeated against the new base. That includes completing a verified branch that
   is only missing its pull request - creating the pull request is still a write.
   Only an exactly matching existing pull request is returned after the base has
   moved, because that path writes nothing.

Writes are a blob per replaced file, one tree, one commit, one ref and one draft
pull request. Each created blob's id is compared with the local object id of the
exact approved bytes, so content that the server stores differently is refused.
GitHub's own tree objects are re-read and verified after creation.

The whole transaction for one candidate and payload is serialised across zt
processes with the same native file lock the credential store uses, held from just
before the first journal write until the pull request is observed. A second
submission waits, then reconciles with what the first actually did instead of
writing duplicates. Journal transitions are monotone: a completed submission is
terminal, and an older attempt cannot overwrite it with an earlier state. A lock
left behind by a crashed process is never stolen - the error names the path so the
operator can confirm it is stale and remove it.

A journal makes an interrupted attempt recoverable and keeps that state outside
the immutable candidate:

```
.zt/artifacts/evolve/submit/<candidate-id>-<public-payload-digest>.json
```

It is written before the first remote write and advanced after each observed step
(`started`, `objects_created`, `ref_created`, `completed`). A transport failure, a
rate limit or a 5xx after a write began is reported as an uncertain outcome, with
the journal path. Recovery is then decided by reading: an exact pull request is
adopted with no writes, a verified branch missing only its pull request gets just
the pull request, and an attempt that created only unreferenced objects simply
completes. The journal holds no token, no log and no private path.

Journal updates are atomic (temporary file plus rename), which guarantees that a
reader never sees a half-written record and that a process interrupted mid-write
leaves the previous one. That is not a durability claim across machine power loss,
and zt does not fsync directory state.

## Status

```sh
zt evolve status CANDIDATE.json --policy policy.json --upstream /path/upstream \
  --validation validation.json --root /path/to/workspace

zt evolve status CANDIDATE.json --policy policy.json --upstream /path/upstream \
  --validation validation.json --root /path/to/workspace --remote --client-id Iv1.example
```

Local status is read-only: it revalidates the candidate, reports the stored
verification record, the submission journal and `publication: not_ready` unless a
completed journal records a draft pull request. It reads no credential and makes
no connection, and it never creates the artifact tree.

`--remote` is the explicit opt-in that reads the pull request state, draft flag,
exact head commit and GitHub-observed check-run counts. Differences are reported
as drift (`head-commit-changed`, `public-text-changed`, `base-branch-moved-past-pinned-base`,
`pull-request-not-draft`, and so on) and are never rewritten or reported as an
exact match. Check-run counts are GitHub's observation, not zt execution.

## Limits and truthful results

- At most eight UTF-8 text files, 256 KiB each and 512 KiB total content.
- Only `skills/<selected-id>/SKILL.md`, `LICENSE`, and Markdown under
  `references/`. Absolute/traversal/backslash/control-character paths, case
  collisions, symlink/submodule targets and file/directory conflicts are refused.
- Policy: 64 KiB; proposal: 1 MiB; canonical candidate: 2 MiB; validation
  contract: 64 KiB; approval: 64 KiB; journal: 64 KiB.
- Verification workspaces: the pinned base tree is capped at 4096 tracked
  entries and read at up to 4 MiB per file and 32 MiB across its unreplaced
  files; candidate replacements add at most eight files and 512 KiB, so a
  materialised workspace is bounded by the base limit plus that candidate
  content rather than by one combined total. Retained logs are bounded per
  command. A pinned base that a particular host filesystem cannot store
  distinctly (case-folding or normalising volumes) is refused before execution
  rather than verified approximately.
- Submission is serialised per candidate and payload by a lock under
  `.zt/artifacts/evolve/locks/`. A process that crashed while holding it leaves the
  lock behind: zt never steals it, and the error names the path for manual cleanup.
- A candidate's own declarations stay `declared_not_verified`, and publication
  stays `not_ready` until a completed submission journal exists. A draft pull
  request is not a merge, a publication or an activation.
- App login and installation do not authorize publication. Approval is a separate
  human decision bound to the exact payload; merging, releasing and activating
  remain outside zt.

Changing policy invalidates the candidate's policy binding. A changed proposal
creates a different candidate; do not edit a stored candidate to preserve an old
approval. Changing the validation contract invalidates both the recorded
verification and the approval. Interrupted preparations may leave identifiable
`.prepare-*` stages; interrupted submissions may leave a journal that records what
may have been created. Inspect them before cleanup rather than treating them as
valid.

Passing this pipeline is not proof of improved Skills. Local verification is
operator-local, the pull request checks are the platform's observation, and a
created draft pull request is not evidence that CI ran or that the change is
correct.
