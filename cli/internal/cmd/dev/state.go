package dev

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

const (
	stateFileName = "dev-listen.json"
	verifyTimeout = time.Second
)

func stateFilePath(configDir string) string { return filepath.Join(configDir, stateFileName) }

// writeState writes the ready object atomically, mode 0600: it names a
// server that answers anyone on loopback.
func writeState(path string, ready devlisten.Ready) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}
	line, err := json.Marshal(ready)
	if err != nil {
		return fmt.Errorf("encoding state file: %w", err)
	}
	tmp, err := writeTemp(dir, append(line, '\n'))
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("renaming state file: %w", err)
	}
	return nil
}

// writeTemp writes data to a new 0600 file in dir and returns its name.
func writeTemp(dir string, data []byte) (string, error) {
	tmp, err := os.CreateTemp(dir, ".dev-listen-*.json")
	if err != nil {
		return "", fmt.Errorf("creating state file: %w", err)
	}
	_, writeErr := tmp.Write(data)
	closeErr := tmp.Close()
	if err := errors.Join(writeErr, closeErr, os.Chmod(tmp.Name(), 0o600)); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("writing state file: %w", err)
	}
	return tmp.Name(), nil
}

func readState(path string) (devlisten.Ready, error) {
	var ready devlisten.Ready
	b, err := os.ReadFile(path)
	if err != nil {
		return ready, fmt.Errorf("reading state file: %w", err)
	}
	if err := json.Unmarshal(b, &ready); err != nil {
		return ready, fmt.Errorf("decoding state file: %w", err)
	}
	return ready, nil
}

// removeStateIfOwned removes the file only while it still names serverID,
// so a server never deletes a newer server's file.
func removeStateIfOwned(path, serverID string) {
	if ready, err := readState(path); err == nil && ready.ServerID == serverID {
		_ = os.Remove(path)
	}
}

// verifyState reports whether the server named by the state file answers
// with the same serverId. pids mean nothing across containers.
func verifyState(ctx context.Context, ready devlisten.Ready) bool {
	ctx, cancel := context.WithTimeout(ctx, verifyTimeout)
	defer cancel()
	client := devlisten.NewClient(ready.URL, devlisten.WithHTTPClient(&http.Client{Timeout: verifyTimeout}))
	info, err := client.Info(ctx)
	return err == nil && info.ServerID == ready.ServerID
}
