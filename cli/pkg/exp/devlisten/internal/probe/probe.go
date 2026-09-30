// Package probe holds the browser probe page and the analytics-js bundle it
// loads. The page sends one track named "dev probe" from a real browser
// SDK to the listener that served it, so a captured probe proves the
// browser-to-listener path.
package probe

import (
	_ "embed"
	"strings"
)

// SDKVersion is the pinned @rudderstack/analytics-js release of the bundle:
// dist/npm/modern/bundled/esm/index.mjs, sha256 92a2b03b...912a712.
const SDKVersion = "3.31.4"

// Published names under /_dev/v1/.
const (
	PagePath    = "probe.html"
	SDKPath     = "probe/analytics-js-" + SDKVersion + ".mjs"
	LicensePath = "probe/LICENSE-analytics-js.md"
)

// Event is the name of the track the page sends.
const Event = "dev probe"

var (
	//go:embed assets/analytics-js-3.31.4.mjs
	sdk []byte
	//go:embed assets/LICENSE-analytics-js.md
	license []byte
)

// The page reads the listener URL from its own origin, so it needs no
// template and works on any port.
const page = `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>rudder-cli dev probe</title></head>
<body>
<h1>rudder-cli dev probe</h1>
<p id="status">Loading @rudderstack/analytics-js {{VERSION}} from this listener.</p>
<p>@rudderstack/analytics-js is under the Elastic License 2.0: <a href="{{LICENSE}}">license</a>.</p>
<script type="module">
import { RudderAnalytics } from './{{SDK}}';
const status = document.getElementById('status');
const origin = location.origin;
const analytics = new RudderAnalytics();
analytics.load('dev', origin, { configUrl: origin, polyfillIfRequired: false });
analytics.ready(() => {
  analytics.track('{{EVENT}}', { probe: true }, () => {
    status.textContent = 'sent: {{EVENT}} went to ' + origin + '. Check it with rudder-cli dev summary.';
    document.title = 'rudder-cli dev probe: sent';
  });
});
</script>
</body>
</html>
`

// Page is the probe page.
func Page() []byte {
	return []byte(strings.NewReplacer("{{VERSION}}", SDKVersion, "{{SDK}}", SDKPath,
		"{{LICENSE}}", LicensePath, "{{EVENT}}", Event).Replace(page))
}

// SDK is the analytics-js bundle.
func SDK() []byte { return sdk }

// License is the analytics-js license, served next to the bundle.
func License() []byte { return license }
