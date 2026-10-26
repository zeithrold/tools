# GitHub App authentication

`zt auth` uses GitHub App Device Flow user tokens and native OS credential
storage. It accepts no App private key, client secret, PAT, `gh auth` state or
environment token. Offline `zt evolve prepare/plan` need no authentication.
Development after `v0.1.1` adds reviewed Draft PR submission through the same
GitHub App and OS-keystore path; independent acceptance and an explicitly
authorized live submission remain separate gates. The published `v0.1.1`
binary does not include that lifecycle. See the [roadmap](roadmap.md).

## Create and install the App

1. Open GitHub Settings → Developer settings → GitHub Apps → New GitHub App.
2. Choose a unique name and informational homepage. Enable **Device Flow** and
   expiring user access tokens. No callback URL, setup URL or client secret is
   needed for this flow.
3. Leave webhooks inactive; the CLI needs no public endpoint.
4. Choose "Only on this account" for personal use, or "Any account" when
   distributing. Install only into explicitly selected repositories.
5. Record the public **Client ID**, not the numeric App ID or installation ID.
   GitHub App client IDs use the `Iv1.`/`Iv23` form; OAuth App `Ov...` IDs are rejected.

The proposed PR workflow needs these repository permissions. Authentication
alone does not require granting future write capabilities.

| Permission | Level | Purpose |
| --- | --- | --- |
| Metadata | Read | Required repository identity |
| Contents | Read/write | Candidate trees, commits and branches |
| Pull requests | Read/write | Draft PR creation and lookup |
| Checks / Actions | Read | Independent results and workflow evidence |
| Issues / Discussions / Workflows / Administration / Secrets / Members | None | Outside the proposed write scope |

Do not add the App to protection bypass lists. GitHub has no permission for
"create PRs but never merge"; require human review through branch protection.
Contents write is not path-scoped. Trusted operator policy must limit targets
and paths, and the proposed adapter must not call merge, auto-merge or deploy.
Installation and login do not authorize a candidate's publication.

## Login and account management

```sh
zt auth login --client-id CLIENT_ID
zt auth status --client-id CLIENT_ID
zt auth status --client-id CLIENT_ID --probe
zt auth logout --client-id CLIENT_ID
```

The public Client ID resolves from `--client-id`, then
`ZEITHROLD_GITHUB_CLIENT_ID`, then the embedded release value. Source builds
must supply it to all four commands unless set in the environment.
`--hostname` defaults to `github.com`; `--profile` defaults to `default`.

- `login` prints a short user code and trusted GitHub verification URL, then
  waits for browser approval. Device codes are not persisted. Polling respects
  GitHub's interval, slowdown, expiry and cancellation.
- Refresh access is requested by default; `--offline-access=false` opts out.
  `--no-browser`, `--no-input` or a non-interactive session suppresses automatic
  browser launch. `--replace` explicitly permits replacing a different account.
- `status` reads local metadata without networking or printing tokens.
  `--probe` verifies remote identity and any selected installation/repository,
  refreshing tokens when necessary.
- `logout` idempotently deletes the selected local record. It does not revoke
  GitHub authorization; revoke that separately in GitHub settings.

User-token access is the intersection of App, user and resource permissions.
Login does not prove installation or repository access; organization approval
or SSO may still be needed. Requests use trusted HTTPS endpoints and do not
forward credentials through cross-host redirects.

### Device authorization rejected

`device_flow_disabled` means the App has not enabled Device Flow. Open GitHub
Settings → Developer settings → GitHub Apps → the relevant App, enable
**Device Flow**, save, and retry `zt auth login`. Changing the local Client ID
does not enable this setting. A valid Client ID alone does not prove the App is
ready for device authorization.

`incorrect_client_credentials` means the public GitHub App Client ID needs to
be checked. Do not substitute the numeric App ID, an OAuth App ID, a client
secret or a PAT. Release builds already embed the public Client ID; use an
explicit override only when deliberately selecting a different App.

Version `v0.1.0` reports non-2xx device-code rejections as an HTTP status without
the OAuth error name. For that version, check Device Flow and the selected
Client ID when the first request returns HTTP 400/401. Newer code retains
allowlisted OAuth error names with local hints, but never displays the remote
description, error URL or raw response body.

## Native credential storage

`99designs/keyring v1.2.2` is restricted to one backend per platform:

| Platform | Required backend |
| --- | --- |
| macOS | Current user's Keychain; CGO and Apple compiler/SDK required |
| Windows | Current user's Credential Manager (WinCred) |
| Linux | Secret Service with an unlocked `login` collection and session bus |

A desktop providing only another collection is not automatically supported.
Locked, denied, unavailable or oversized storage fails without a plaintext,
environment or alternate-account fallback. A readiness check cannot guarantee a
later write succeeds.

Access/refresh tokens and account identity occupy one bounded record, selected
by host, Client ID and profile. Refresh replaces the token pair atomically.
Tokens are never written to project configuration, artifacts, argv or logs, or
exported to candidate processes. Storage protection does not remove tokens from
application memory. macOS items require an unlocked Keychain and disable sync.

Mutations use a non-secret user-cache lock; locks are not stolen based on age.
After a crash, confirm no process owns a reported orphan before manual cleanup.
Linux's upstream package may connect to D-Bus and start `dbus-launch` during
initialization. Inventory commands still do not call credential methods,
authenticate, run project checks or rewrite project files.

## Release configuration and verification

Release CI reads only `ZEITHROLD_GITHUB_CLIENT_ID` and embeds the public value;
never supply a private key, client secret or token as build configuration.

Verify native storage, real browser login, identity, refresh/revocation and
target access separately. A Linux test with synthetic credentials proves
neither macOS/Windows storage nor real GitHub authentication. Future PR acceptance
must also verify exact-head checks and branch protection; creating a PR is not
proof that CI ran.

## References

- [Register an App](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/registering-a-github-app)
- [Install an App](https://docs.github.com/en/apps/using-github-apps/installing-your-own-github-app)
- [Device Flow user tokens](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app)
- [Refresh user tokens](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/refreshing-user-access-tokens)
