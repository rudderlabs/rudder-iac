package telemetry

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

// sessionIdle is how long the CLI can go unused before the next command starts
// a new session.
const sessionIdle = 30 * time.Minute

// localState is kept next to the config file and never sent. Salt keys the
// project hash, so a project_id cannot be reversed by hashing common paths.
type localState struct {
	Salt      string    `json:"salt"`
	SessionID string    `json:"sessionId"`
	LastSeen  time.Time `json:"lastSeen"`
}

// stateFile is a variable so tests can keep the state out of the config directory.
var stateFile = func() string {
	return filepath.Join(config.GetConfigDir(), "telemetry.json")
}

// correlationIDs returns the session_id and project_id of a command run at now
// against location, and records the run in the state file at path.
//
// A session is a run of commands on this machine with no gap longer than
// sessionIdle. A coding agent starts a new shell for every command, so the
// commands share no parent process, terminal or environment to key on; the
// time they run at is what they share.
// ponytail: two agents or terminals working at the same time on one machine
// share a session_id. Split them by project_id, or key on an id the agent
// exports if that is not enough.
func correlationIDs(path, location string, now time.Time) (sessionID, projectID string) {
	var state localState
	if data, err := os.ReadFile(path); err == nil {
		// An unreadable file leaves state empty and both ids start again.
		_ = json.Unmarshal(data, &state)
	}

	if state.Salt == "" {
		state.Salt = uuid.NewString()
	}
	if state.SessionID == "" || now.Sub(state.LastSeen) > sessionIdle {
		state.SessionID = uuid.NewString()
	}
	state.LastSeen = now

	// Without the write the ids last for this command only, which still sends
	// nothing it should not.
	_ = writeState(path, state)

	mac := hmac.New(sha256.New, []byte(state.Salt))
	mac.Write([]byte(projectRoot(location)))

	return state.SessionID, hex.EncodeToString(mac.Sum(nil))
}

// writeState replaces the file in one rename, so a command running at the same
// time never reads half a file and throws the salt away.
func writeState(path string, state localState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "telemetry-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), path)
}

// projectRoot returns the directory that stands for the project: the git
// checkout holding location, or location itself outside a checkout. An empty
// location means the working directory. Commands without a location flag and
// commands pointed at a subdirectory of the same checkout then agree.
func projectRoot(location string) string {
	dir, err := filepath.Abs(location)
	if err != nil {
		return location
	}
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}

	for root := dir; ; root = filepath.Dir(root) {
		if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
			// A location that does not exist yet was not resolved above.
			if resolved, err := filepath.EvalSymlinks(root); err == nil {
				return resolved
			}
			return root
		}
		if root == filepath.Dir(root) {
			return dir
		}
	}
}
