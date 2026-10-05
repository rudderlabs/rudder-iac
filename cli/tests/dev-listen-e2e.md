# dev listen end-to-end coverage

What the end-to-end tests of `rudder-cli dev listen` check, where each one runs, and what no job can check.

## Runs on every pull request (`dev-listen-e2e.yml`)

| Test | File | What it proves |
| --- | --- | --- |
| `TestDevListenRefusesForeignHostsAndCrossSiteReads` | `command_dev_listen_hardening_test.go` | 13 Host forms and 4 `Sec-Fetch-Site` values against the query API and the review page. Ingestion stays open to any Host. |
| `TestDevListenCORSCoversIngestionOnly` | same | CORS headers appear on ingestion and never on `/_dev/v1` or `/_dev/ui/`. |
| `TestDevListenReviewPageIsLockedDown` | same | CSP without `unsafe-inline`, `nosniff`, `no-store`, and 404 for five path traversal forms. |
| `TestDevListenMasksTheWriteKey` | same | A key sent by Basic auth, query string and pixel never appears in `/requests`, `/info` or the guide. Cookies are not stored. |
| `TestDevListenKeepsTheBytesTheSDKSent` | same | Large integers, long decimals, key order, escapes and unicode come back unchanged. |
| `TestDevListenEnforcesBodyAndEventLimits` | same | 2,048,000-byte cap at and over the limit, a 50 MB gzip bomb, 10,000 and 10,001 events, corrupt gzip, truncated JSON, empty body, 600 KiB header. |
| `TestDevListenCapsInFlightRequests` | same | Eight stalled uploads fill the slots, the ninth request gets 503, the query API still answers, and the slots return within 20 s. |
| `TestDevListenDropsASlowHeader` | same | A header that never ends is closed by the server. |
| `TestDevListenCapturesParallelWritersExactly` | same | 150 requests from six writers are all captured, with strictly increasing sequence numbers. |
| `TestDevListenEvictsOldestRequestsAndSaysSo` | same | After 10,050 requests the store holds at most 10,000 and reports `evicted` and `evictedThrough`. |
| `TestDevListenKeepsCapturesWhenOversizedPostsArrive` | same | 40 refused 2 MB posts must not push out earlier captures. **Fails today** because #936 keeps the full body of a refused request. The pull request that adds it stays a draft until #936 truncates refused bodies. |
| `hostile events` (3 tests) | `e2e-ui/tests/xss.spec.js` | Seven script and markup payloads in every rendered field never run and never become elements. The page requests nothing outside `/_dev/`. |

The Go tests take about 17 s with `-race`. They need no network, account or secret.

## Runs nightly (`dev-listen-nightly.yml`)

- The Go suite five times under the race detector. Repeats find slot leaks and port races that one pass hides.
- Ten minutes of fuzzing on each of the two fuzz targets, instead of 30 s.
- The hostile event test in Chromium, Firefox and WebKit. A CSP or `Sec-Fetch-Site` difference shows up only in the engine that has it.

## Not covered by any job

| Gap | Why it stays manual |
| --- | --- |
| DNS rebinding with a real DNS server | Needs a name that answers 127.0.0.1 after the page loads. The Host tests send the forged header directly, which covers the server check but not the browser. |
| Chrome Private Network Access preflight from a public HTTPS page | Behaviour depends on the Chrome version and a flag. Check by hand when Chrome ships a change. |
| iOS and Android SDKs, Unity, Flutter, React Native | Need simulators or devices. The guide lists the `controlPlaneUrl` setting. Check once per SDK major release. |
| Ruby, PHP, Java SDKs | Work, but need three more runtimes in CI for a setting the guide already states. Add if a customer reports a break. |
| Real memory use at the 64 MiB cap | The cap is an estimate of retained bytes. Resident memory depends on the Go runtime and the runner, so a threshold would be flaky. |
| Windows and macOS binding, IPv6 `localhost` | The CI runners are Linux. Run `make test-e2e` on a Mac before a release. |
| A laptop that sleeps while a listener runs | Timers pause during suspend and the 10 s body deadline appears to stretch. Cannot be scripted on a hosted runner. |
| Cross-container Host handling with a Compose service name | The container test covers one container. A multi-container test needs Compose on the runner and adds little over `TestDevListenChecksTheHostOnAWildcardBind`. |
