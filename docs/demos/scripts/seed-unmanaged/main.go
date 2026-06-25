// seed-unmanaged creates (or deletes) a genuinely UNMANAGED event-stream source
// directly via the api client — a source with NO external id, as if someone made
// it in the dashboard. It exists to stage `rudder-cli set-external-id` demos
// (adopting an unmanaged remote resource), which the CLI itself can't set up
// since every CLI-created source gets an external id.
//
// The create endpoint rejects an *empty* externalId but accepts an *omitted*
// one, so we POST a body without the field at all.
//
//	go run ./docs/demos/scripts/seed-unmanaged                 # create unmanaged, prints remote id
//	go run ./docs/demos/scripts/seed-unmanaged -name "Legacy Orders"
//	go run ./docs/demos/scripts/seed-unmanaged -ext-id foo     # create already-managed (with ext id)
//	go run ./docs/demos/scripts/seed-unmanaged -delete <id>    # cleanup
//
// Auth/URL come from ~/.rudder/config.json (same as the CLI).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// Mirrors source.prefix in api/client/event-stream/source (unexported there).
const sourcesPath = "/v2/event-stream-sources"

type rudderConfig struct {
	APIURL string `json:"apiURL"`
	Auth   struct {
		AccessToken string `json:"accessToken"`
	} `json:"auth"`
}

func main() {
	name := flag.String("name", "Demo Unmanaged Source", "source name")
	srcType := flag.String("type", "javascript", "source type")
	// Empty (default) => omit externalId => the source is created UNMANAGED.
	extID := flag.String("ext-id", "", "external id to create with; empty omits it (unmanaged)")
	del := flag.String("delete", "", "delete the source with this remote id, then exit")
	flag.Parse()

	home, err := os.UserHomeDir()
	if err != nil {
		fail("resolving home dir: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".rudder", "config.json"))
	if err != nil {
		fail("reading ~/.rudder/config.json: %v", err)
	}
	var cfg rudderConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		fail("parsing config: %v", err)
	}
	if cfg.Auth.AccessToken == "" {
		fail("no access token in config; run `rudder-cli auth login` first")
	}

	opts := []client.Option{}
	if cfg.APIURL != "" {
		opts = append(opts, client.WithBaseURL(cfg.APIURL))
	}
	c, err := client.New(cfg.Auth.AccessToken, opts...)
	if err != nil {
		fail("creating api client: %v", err)
	}
	ctx := context.Background()

	if *del != "" {
		if _, err := c.Do(ctx, "DELETE", sourcesPath+"/"+*del, nil); err != nil {
			fail("deleting source %s: %v", *del, err)
		}
		fmt.Printf("deleted source %s\n", *del)
		return
	}

	payload := map[string]any{"name": *name, "type": *srcType, "enabled": true}
	if *extID != "" { // omit when empty => unmanaged
		payload["externalId"] = *extID
	}
	body, _ := json.Marshal(payload)
	resp, err := c.Do(ctx, "POST", sourcesPath, bytes.NewReader(body))
	if err != nil {
		fail("creating source: %v", err)
	}
	var r struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		ExternalID string `json:"externalId"`
	}
	if err := json.Unmarshal(resp, &r); err != nil {
		fail("parsing response: %v", err)
	}
	fmt.Printf("created source: id=%s name=%q externalId=%q\n", r.ID, r.Name, r.ExternalID)
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "seed-unmanaged: "+format+"\n", a...)
	os.Exit(1)
}
