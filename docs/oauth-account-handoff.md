# The OAuth account browser handoff API

This document is the contract between `rudder-cli` and the control plane for
creating and re-authorizing an OAuth integration account. The CLI half is
built (`cli/internal/oauthhandoff`, `rudder-cli workspace accounts create`,
`rudder-cli workspace accounts reauthorize`). **Both halves are released.**
rudder-control-plane#2806, rudder-webapp#10426 and rudder-config-backend#6972
have all merged and are live in production (config-backend 2.100.0, rudder-api
v0.48.0). The commands still sit behind an experimental flag, and the consent
page is behind the webapp's own `enable-cli-account-handoff` flag, which no CLI
setting turns on: with that flag off for a workspace the page refuses, and the
refusal comes back to the waiting terminal.

Read this as the specification to implement. Every request, response, status
value and timeout the CLI depends on is stated here. Nothing in it requires
reading Go.

## Why a server-held session instead of a loopback redirect

The usual CLI OAuth dance starts a loopback listener on `127.0.0.1` and makes
it the OAuth `redirect_uri`. That cannot work here.

`rudder-auth` owns the `redirect_uri` sent to the provider. The config
backend's `OAuthService.getOAuthURLV1` takes the `Location` of rudder-auth's 302
as the authorize URL and never chooses the redirect target itself. A URL on the user's machine can therefore not be the redirect target
without a change in rudder-auth and in every provider's registered redirect
list.

Creating an OAuth account is also user-attributed. `callBackV1` needs a
`userId` and the `credentialsManagement` role. The CLI's workspace access
token carries neither, so the consent round trip has to run under the user's
own dashboard session in a browser.

So the browser does the whole OAuth exchange exactly as the dashboard does it
today, and the CLI watches a session the server holds. The CLI never sees a
provider token, a `code`, or a redirect.

## Shape of the flow

```
CLI                         config backend                 browser (user session)
 |  POST /v2/oauth/cli-sessions  |                                 |
 |------------------------------>|  creates session, status=pending
 |  201 {sessionId, webUrl}      |                                 |
 |<------------------------------|                                 |
 |  prints webUrl, opens browser ----------------------------------->  GET webUrl
 |                               |                                 |  (existing oauth/url,
 |  GET  .../{id}?wait=30s       |                                 |   provider consent,
 |------------------------------>|  hangs while pending            |   existing oauth/callback)
 |                               |<--- POST .../{id}/result -------|
 |  200 {status: complete, ...}  |  status=complete                |
 |<------------------------------|                                 |
 |  writes the account spec, naming the new account as its remote id
 |  ('rudder-cli apply' then adopts it, which is what claims the id)
```

## Endpoints

Three endpoints are new, written here as the CLI sees them: on the public API
host, alongside `/v2/accounts`.

**Two services carry them.** `/v2/...` is served by `rudder-api`, which
resolves the workspace from the access token and calls the config backend at
`/apigateway/v1/workspaces/{workspaceId}/...`. That is also where the auth
scheme changes: `rudder-api` takes the `Bearer` token, resolves it, and calls
the config backend with basic auth, which is what `authenticateAccessToken`
reads. So the CLI sends `Bearer` and nothing about it has to change, but the
`/v2/oauth/cli-sessions` routes have to exist in **both** services. The config
backend half is implemented in rudderlabs/rudder-config-backend#6972 and the
`rudder-api` half in rudderlabs/rudder-control-plane#2806, which proxies the two
routes the CLI calls. Only those two are public: the consent page's own read and
result routes are the dashboard talking to the config backend directly, under
the signed-in user's session, and putting them on the public API would expose a
session to any workspace token rather than to the user whose consent it is.

### 1. Open a session

```
POST /v2/oauth/cli-sessions
Authorization: Bearer <workspace access token>
Content-Type: application/json
```

The body is one of two shapes, and the discriminator is the presence of
`accountId`. This mirrors what the backend already does for the OAuth callback:
`isCreateOAuthAccountRequest` is `!('accountId' in request)`
(`oauth.service.ts:61`).

Create:

```json
{ "accountDefinitionName": "DESTINATION_GOOGLE_ADWORDS_OFFLINE_CONVERSIONS_OAUTH",
  "name": "Google Ads (prod)" }
```

Re-authorize:

```json
{ "accountId": "2fS9kQwPzX7rTnV4yLbH8mJcD1e" }
```

| Field | Create | Re-authorize |
| --- | --- | --- |
| `accountDefinitionName` | required, must name an existing OAuth account definition | must be absent |
| `name` | required, the account's display name | must be absent |
| `accountId` | must be absent | required, an OAuth account in the token's workspace |

Response `201 Created`:

```json
{ "sessionId": "9f3a2c1e",
  "webUrl": "https://app.rudderstack.com/oauth/cli/9f3a2c1e",
  "expiresAt": "2026-11-17T12:34:56Z" }
```

- `sessionId` is opaque, URL-safe, and unguessable (at least 128 bits of
  entropy). It is the only credential the browser page needs, so it must not be
  enumerable.
- `webUrl` is an absolute URL to the webapp page that drives consent. The CLI
  prints it verbatim and opens it in a browser. It must be openable in any
  browser signed in to RudderStack, not only on the machine running the CLI.
- `expiresAt` is RFC 3339 UTC.

Errors:

| Status | When |
| --- | --- |
| `400` | both or neither of `accountDefinitionName` and `accountId`; a missing `name` on create; an `accountDefinitionName` that is not an OAuth account definition |
| `401` | missing or invalid access token |
| `404` | `accountId` is not an account in the token's workspace |
| `409` | `accountId` names a non-OAuth account (its credentials are not held by rudder-auth, so there is nothing to re-authorize) |

A session is bound to the workspace of the access token that opened it. The
browser user must be a member of that workspace; see "What the webapp owes".

### 2. Poll a session

```
GET /v2/oauth/cli-sessions/{sessionId}?wait=30s
Authorization: Bearer <workspace access token>
```

`wait` is an optional Go duration string, `0s` to `2m0s`, default `30s`. A
value above the maximum is clamped rather than refused; a value that is not a
duration is a `400`. Parse the full Go format rather than a seconds suffix:
the CLI sends what `time.Duration.String()` emits, which is `2m0s` rather than
`120s`, and `29.8s` when it trims a poll to the time it has left. While the session is
`pending` the server holds the request open until the status changes or `wait`
elapses, then answers with the current state. Any other status answers
immediately.

Response `200 OK`:

```json
{ "status": "pending",
  "accountId": "",
  "accountDefinitionName": "DESTINATION_GOOGLE_ADWORDS_OFFLINE_CONVERSIONS_OAUTH",
  "name": "Google Ads (prod)",
  "error": "" }
```

| Field | Meaning |
| --- | --- |
| `status` | `pending`, `complete`, `denied` or `expired`. An unknown value is an error to the CLI, so new statuses are a breaking change. |
| `accountId` | the account's remote id once `status` is `complete`; empty otherwise. On a re-authorize session it is the incoming account id from the start, and it does not change. |
| `accountDefinitionName` | echoed from the opening request, so a caller that has lost track of what it asked for can still name it |
| `name` | echoed from the opening request, same reason |
| `error` | on `denied`, a sentence to show the user; empty otherwise |

Errors: `401` for a bad token, and `404` for an unknown session, a purged one,
or one belonging to another workspace. A mismatch is deliberately not a `403`:
with the workspace resolved from the token, "a session by that id exists, but
not yours" is more than the caller should learn. The CLI treats both alike.

The session must be readable by any valid access token for its workspace, not
only by the exact token that opened it: a user may re-run the command in
another shell.

### 3. Report the result (webapp to backend)

```
POST /oauth/cli-sessions/{sessionId}/result
```

Called by the webapp page under the user's own dashboard session, never by the
CLI. It is not a `/v2` route: the page is served by the dashboard and talks to
the config backend directly, carrying only the session id, so there is no
workspace in the path and no access token involved. Body on success:

```json
{ "accountId": "2fS9kQwPzX7rTnV4yLbH8mJcD1e" }
```

Body on refusal:

```json
{ "error": "Your RudderStack user can't create accounts in this workspace. Creating an integration account needs the credentialsManagement role." }
```

Response `204 No Content`.

The first call wins. A second call against a session that is no longer
`pending` is a no-op and still answers `204`, so a double-submitted page cannot
create two accounts or overwrite a recorded result.

On a create session the reported `accountId` must belong to the account
definition the session was opened with, and to the session's workspace. On a
re-authorize session it must equal the session's own `accountId`; anything else
is `400`.

## Status transitions

```
            browser reports an accountId
pending  ───────────────────────────────────>  complete
   │
   │        browser reports an error
   ├───────────────────────────────────────>  denied
   │
   │        expiresAt passes while pending
   └───────────────────────────────────────>  expired
```

`complete`, `denied` and `expired` are terminal. No transition leaves a
terminal state, and a terminal state's fields never change afterwards.

The CLI does not re-read a session it has stopped waiting for. A consent that
finishes late has created an account that nothing manages, and the ordinary
import path adopts it: `rudder-cli import workspace` writes its spec and
`rudder-cli apply` claims the id. The stickiness still matters to the server
half, which has to answer a second report against a resolved session without
creating a second account.

## Timeouts and retention

| Bound | Value | Why |
| --- | --- | --- |
| Session TTL while `pending` | 10 minutes, reported as `expiresAt` | long enough for a provider consent screen with a sign-in and an MFA prompt |
| Retention after a terminal status | at least 1 hour; the implementation keeps a day | the window in which a result can still be read, by a second report or by an operator |
| Long-poll `wait` | `0s`–`2m0s`, default `30s` | the default stays under a typical 60s proxy idle timeout. Confirm the ingress idle timeout before raising it past 60s: Node's own `keepAliveTimeout` and `headersTimeout` do not bound a response in progress, so the edge is what decides |
| Gap between two polls | at least 500ms, enforced by the CLI | the server answers `pending` at once when less than its own re-read interval is left of `wait`, so without a floor the tail of a wait would be back-to-back session reads |
| CLI wait, end to end | `--timeout`, default 5 minutes | the CLI stops waiting and exits non-zero, saying how to adopt an account a finished consent may have created; the session is untouched |

**A failed poll ends the wait.** One non-JSON `503`, one `429` or one dropped
connection stops the CLI waiting, rather than being retried: before consent
nothing exists and the command can simply be run again, and after consent the
account exists and the import path adopts it. There is no state the CLI has to
get back to, so there is no retry loop here to get wrong.

The CLI's `--timeout` is independent of the session TTL. A CLI timeout does not
cancel the session.

## What the CLI does that needs nothing new

Claiming the account under its spec id is `PUT /v2/accounts/{id}/external-id`,
which is already open to the CLI and already idempotent (`setExternalId` in the
config backend's account service). The public account API refuses `POST` create
and `PUT` update for an OAuth account (`createNonOAuthAccount` and
`editAccount`), which is why the create path has to go through this handoff at
all.

`workspace accounts create` does not call it. The account it creates belongs to
no project, exactly as an account made in the dashboard does, and the claim
happens where every other adoption happens: in `apply`. With `--write-spec` the
command writes the spec that names the new account, carrying the same import
metadata `import workspace` writes:

```yaml
version: "rudder/v1"
kind: "account"
metadata:
  import:
    workspaces:
      - resources:
          - remote_id: "2abcDEFghiJKL"
            urn: "account:google-ads-prod"
        workspace_id: "2xyzWORKSPACE"
  name: "account"
spec:
  account_definition_name: "DESTINATION_GOOGLE_ADWORDS_OFFLINE_CONVERSIONS_OAUTH"
  id: "google-ads-prod"
  name: "Google Ads (prod)"
```

That block is what makes the next `apply` adopt the account rather than try to
create it: the differ treats a local resource with import metadata for this
workspace as importable, and the accounts handler's import of an OAuth account
is the external-id claim alone. Without `--write-spec`, `import workspace`
writes the same spec, and its id comes from the same kebab-case namer, so both
routes end in the same place.

The ordering matters for what a failure leaves behind. Nothing the CLI does
after consent changes the workspace, so a failed spec write, a lost connection
or a Ctrl-C leaves an unmanaged account and nothing else, which the import path
picks up. An account claimed with no spec beside it would instead be a managed
resource the project does not declare, which `apply` plans to delete.

## Where the server half is

All three services are implemented. Two have merged.

rudderlabs/rudder-config-backend#6972, still open, holds the session itself: the
four endpoints, a Postgres `oauth_cli_sessions` table, the TTLs, the terminal
statuses, first-call-wins on the result, and the webapp twin of the external-id
route the dashboard needs. rudderlabs/rudder-control-plane#2806, merged, proxies
the two public ones from `/v2` onto it. rudderlabs/rudder-webapp#10426, merged,
serves the consent page.

None of them changes an existing OAuth path. The create and re-auth flows behind
the webapp's calls are reused as they are: the re-auth branch of `callBackV1`
already updates an existing account's secret in place and returns the same
account id, so re-authorize is a payload difference rather than a new mechanism.

The chain is proven end to end against a real consent screen. On 2026-11-24,
with rudder-cli 1.4.0 against production, a create handoff reached `complete`
and created an account, `apply` adopted it (`externalId` went from null to the
spec id), a second `apply` reported no changes, and `reauthorize` replaced the
secret in place: same account id, same `externalId`, `createdAt` unchanged and
only `updatedAt` moved. The re-authorize claim that no spec and no destination
needs editing held.

### The poll does not have to wake on an event

An earlier draft of this document asked for a long-poll that wakes on a state
change rather than on a timer. That was a mechanism where a latency bound
belongs, and the config backend has nothing to build it on: no pub/sub, an
in-process array for message relay, and one SSE endpoint that only proxies an
upstream stream. With several pods, the pod holding the CLI's poll is almost
never the pod the browser posts the result to.

The requirement is therefore latency, not mechanism: a terminal status must
reach a waiting CLI within a second of being written. #6972 does it by
re-reading the session row every 500 ms inside the held request, which costs
one primary-key read per waiter per interval, on a call that happens once per
`accounts create`. That is indistinguishable from an event at the pace of a
human finishing a consent screen.

## What the webapp still owes

One page at `webUrl`, which #6972 builds as
`${webAppURL}/oauth/cli/<sessionId>`, so the webapp has to serve that path. It
reads and reports through two routes of its own, reached by session id with no
workspace in the path, under the user's dashboard session:
`GET /oauth/cli-sessions/{sessionId}` and endpoint 3 below. The page:

1. Reads the session by id and shows which account definition or account it is
   about. A user outside the session's workspace is refused with a `403` there,
   which is the refusal to report back to the terminal.
2. Refuses with a clear message, reported through endpoint 3, when the signed-in
   user is not a member of the session's workspace, or lacks the
   `credentialsManagement` role that `callBackV1` requires. Surfacing the
   denial here rather than from the CLI's token is deliberate: the CLI's token
   is not the identity being checked.
3. Runs today's dashboard flow unchanged: `POST oauth/url`, the provider's
   consent screen, the redirect back to rudder-auth, `POST oauth/callback`.
4. Reports the resulting `accountId`, or the error, to endpoint 3, and tells the
   user to return to their terminal.

## Not covered by this contract

`linkedin_ads` and `bingads_offline_conversions` cannot be reached through this
flow. Neither has an account definition in integrations-config; their accounts
come from the legacy role-keyed OAuth flow
(`GET /oauth/:rudderCategory/:role/url`) and carry a null
`accountDefinitionName`. A handoff session has no definition name to open with,
so `workspace accounts create` refuses those by name before any request. Giving
them an account definition is integrations-config work owned by the destination
teams, and it is a prerequisite for CLI coverage rather than a part of this
contract.
