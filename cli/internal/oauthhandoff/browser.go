package oauthhandoff

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// CanOpenBrowser reports whether this host has a browser to open. It is false
// on a headless Linux box, where neither a display nor a $BROWSER override is
// set, so the CLI can print the link and wait instead of pretending to launch
// something. macOS and Windows always have an opener.
func CanOpenBrowser() bool {
	return canOpenBrowser(runtime.GOOS, os.Getenv)
}

func canOpenBrowser(goos string, env func(string) string) bool {
	if goos != "linux" {
		return true
	}
	// WAYLAND_DISPLAY covers a Wayland session, which has no $DISPLAY unless
	// XWayland is running.
	for _, key := range []string{"DISPLAY", "WAYLAND_DISPLAY", "BROWSER"} {
		if env(key) != "" {
			return true
		}
	}
	return false
}

// OpenBrowser points the host's default browser at url.
//
// ponytail: three exec lines and no dependency. The URL is printed before this
// runs, so a failed launch is a message rather than a failed command.
func OpenBrowser(url string) error {
	// Every URL that reaches here has been through safeWebURL. This is the
	// second belt: each opener below takes its argument as a scheme handler to
	// launch, so a caller that skipped the check must not reach exec. Checked
	// before the host has an opener at all, so the refusal does not depend on
	// where this runs.
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return fmt.Errorf("refusing to open %q: not an http URL", url)
	}
	if !CanOpenBrowser() {
		return errors.New("no browser on this host")
	}

	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		if browser := os.Getenv("BROWSER"); browser != "" {
			return exec.Command(browser, url).Start()
		}
		return exec.Command("xdg-open", url).Start()
	}
}
