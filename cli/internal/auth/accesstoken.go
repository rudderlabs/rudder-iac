package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"golang.org/x/term"
)

type LoginOptions struct {
	NoBrowser  bool
	JSONOutput bool
	UserAgent  string
	Stdout     io.Writer
	Stderr     io.Writer
}

// Login signs in through the browser, stores the token in the keychain and
// prints the workspace the token belongs to.
func Login(ctx context.Context, opts LoginOptions) error {
	var (
		cfg        = config.GetConfig()
		httpClient = &http.Client{Timeout: 30 * time.Second}
	)

	grant, err := BrowserLogin(ctx, BrowserLoginOptions{
		AppURL:      cfg.AppURL,
		APIURL:      cfg.APIURL,
		ClientName:  clientName(),
		NoBrowser:   opts.NoBrowser,
		Timeout:     5 * time.Minute,
		Progress:    opts.Stderr,
		OpenBrowser: OpenBrowser,
		HTTPClient:  httpClient,
	})
	if err != nil {
		return err
	}

	if err := NewKeychain().Set(KeychainAccount(cfg.APIURL), grant.AccessToken); err != nil {
		return fmt.Errorf("storing token in keychain (use `rudder-cli auth login --with-token` where no keychain exists): %w", err)
	}
	config.SetKeychainLogin(grant.WorkspaceID, grant.WorkspaceName)

	c, err := client.New(grant.AccessToken, client.WithBaseURL(cfg.APIURL), client.WithUserAgent(opts.UserAgent))
	if err != nil {
		return fmt.Errorf("creating API client: %w", err)
	}
	workspace, err := c.Workspaces.GetByAuthToken(ctx)
	if err != nil {
		return fmt.Errorf("fetching workspace information: %w", err)
	}

	if opts.JSONOutput {
		return json.NewEncoder(opts.Stdout).Encode(map[string]string{
			"status":        "authenticated",
			"workspaceID":   workspace.ID,
			"workspaceName": workspace.Name,
			"tokenName":     grant.TokenName,
			"tokenStore":    config.TokenStoreKeychain,
		})
	}
	_, err = fmt.Fprintf(opts.Stdout, "Signed in to workspace %q (%s). The token is in the OS keychain.\n", workspace.Name, workspace.ID)
	return err
}

// clientName names the token after this machine. RUDDERSTACK_CLI_DEVICE_NAME
// overrides the hostname, for shared or generically named machines.
func clientName() string {
	if name := strings.TrimSpace(os.Getenv("RUDDERSTACK_CLI_DEVICE_NAME")); name != "" {
		return "rudder-cli on " + name
	}
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "rudder-cli"
	}
	return "rudder-cli on " + host
}

// LoginWithToken stores a token in the config file, for CI and hosts without
// a keychain. On a terminal it prompts without echo; otherwise it reads the
// first line of stdin.
func LoginWithToken(stdin *os.File) error {
	var (
		accessToken string
		err         error
	)
	if term.IsTerminal(int(stdin.Fd())) {
		accessToken, err = ui.AskSecret("Enter your access token:")
	} else {
		accessToken, err = readTokenLine(stdin)
	}
	if err != nil {
		return fmt.Errorf("reading access token: %w", err)
	}
	if accessToken == "" {
		return errors.New("access token is empty")
	}

	// A keychain token outranks the file, so drop it to make this one current.
	removeKeychainToken(config.GetConfig().APIURL)
	config.SetFileToken(accessToken)
	return nil
}

func readTokenLine(r io.Reader) (string, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// Logout removes the keychain token and every auth key from the config file.
// A RUDDERSTACK_ACCESS_TOKEN in the environment stays in effect.
func Logout(stderr io.Writer) error {
	removeKeychainToken(config.GetConfig().APIURL)
	config.ClearAuth()

	fmt.Fprintln(stderr, "Signed out. The token stays valid until you revoke it in the RudderStack app.")
	if os.Getenv(config.AccessTokenEnvVar) != "" {
		fmt.Fprintf(stderr, "%s is still set in your environment.\n", config.AccessTokenEnvVar)
	}
	return nil
}

func removeKeychainToken(apiURL string) {
	err := NewKeychain().Delete(KeychainAccount(apiURL))
	if err != nil && !errors.Is(err, ErrTokenNotFound) {
		log.Warn("removing token from keychain", "error", err)
	}
}
