package agentinit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/pretty"
	"github.com/tidwall/sjson"
)

// Actions reported for each path init touches.
const (
	actionCreated   = "created"
	actionUpdated   = "updated"
	actionUnchanged = "unchanged"
	actionSkipped   = "skipped"
)

const (
	gitDir          = ".git"
	specDir         = "rudder"
	specReadme      = "rudder/README.md"
	gitignoreFile   = ".gitignore"
	varFilePattern  = "rudder/*.vars.yaml"
	mcpFile         = ".mcp.json"
	mcpServerName   = "rudderstack"
	mcpServerURL    = "https://mcp.rudderstack.com/bearer-auth-mcp"
	agentsFile      = "AGENTS.md"
	claudeFile      = "CLAUDE.md"
	agentsImport    = "@AGENTS.md"
	agentsBlockOpen = "<!-- rudderstack:begin -->"
	agentsBlockEnd  = "<!-- rudderstack:end -->"
	gitMissingNote  = "git is not installed"
)

const specReadmeContent = "# RudderStack specs\n" +
	"\n" +
	"The RudderStack specs for this project live in this directory. `rudder-cli` reads\n" +
	"every YAML file under the directory it is given, except `*.vars.yaml` files, so\n" +
	"pass `-l rudder` to every command:\n" +
	"\n" +
	"    rudder-cli validate -l rudder --var-file rudder/secrets.vars.yaml\n" +
	"    rudder-cli apply -l rudder --var-file rudder/secrets.vars.yaml --dry-run\n" +
	"\n" +
	"## Secrets\n" +
	"\n" +
	"Put secrets in `rudder/secrets.vars.yaml`, one `KEY: value` per line:\n" +
	"\n" +
	"    WEBHOOK_AUTH: Bearer <token>\n" +
	"\n" +
	"Reference them in specs as `'{{ .KEY }}'`, for example `'{{ .WEBHOOK_AUTH }}'`,\n" +
	"and pass `--var-file rudder/secrets.vars.yaml` to `validate` and `apply`. The\n" +
	"`rudder/*.vars.yaml` line in `.gitignore` keeps the file out of git.\n"

// agentsBlockBody takes the workspace name and ID.
const agentsBlockBody = "## RudderStack\n" +
	"\n" +
	"This project sends events to RudderStack workspace '%s' (%s).\n" +
	"\n" +
	"To set it up, follow the rudder-onboarding skill. If it is not installed, run\n" +
	"`npx skills add rudderlabs/rudder-agent-skills -y -s rudder-onboarding`, or read\n" +
	"<https://github.com/rudderlabs/rudder-agent-skills/blob/main/plugins/rudder-core/skills/rudder-onboarding/SKILL.md>.\n" +
	"\n" +
	"Rules:\n" +
	"\n" +
	"- Specs live in `rudder/`. Pass `-l rudder` to every `rudder-cli` command.\n" +
	"- Show the user the output of `rudder-cli apply -l rudder --dry-run` and get their yes\n" +
	"  before running `rudder-cli apply -l rudder --confirm=false`.\n" +
	"- Never apply a plan that lists `Removed resources:` without the user's explicit approval.\n" +
	"  `apply` reconciles every CLI-managed resource in the workspace, not only the specs in `rudder/`.\n" +
	"- Secrets go in `rudder/secrets.vars.yaml`, never in specs or chat.\n" +
	"- The RudderStack MCP server is configured in `.mcp.json` and uses the same access token\n" +
	"  as `rudder-cli`."

type fileAction struct {
	Path   string `json:"path"`
	Action string `json:"action"`
	Note   string `json:"note,omitempty"`
}

func (a fileAction) String() string {
	line := fmt.Sprintf("%-10s %s", a.Action, a.Path)
	if a.Note != "" {
		line += " (" + a.Note + ")"
	}
	return line
}

// prepare brings dir to the state an agent needs. It returns the actions taken
// so far along with any error, so the caller can report a partial run.
func prepare(ctx context.Context, dir, workspaceID, workspaceName string) ([]fileAction, error) {
	var actions []fileAction

	git, err := ensureGitRepo(ctx, dir)
	if err != nil {
		return actions, err
	}
	actions = append(actions, git)

	specs, err := ensureSpecDir(dir)
	if err != nil {
		return actions, err
	}
	actions = append(actions, specs)

	edits := []struct {
		path string
		edit editFunc
	}{
		{specReadme, writeIfMissing([]byte(specReadmeContent))},
		{gitignoreFile, addGitignoreLine},
		{mcpFile, mergeMCPServer(workspaceID)},
		{agentsFile, upsertAgentsBlock(agentsBlock(workspaceID, workspaceName))},
		{claudeFile, importAgents},
	}
	for _, e := range edits {
		a, err := editFile(dir, e.path, e.edit)
		if err != nil {
			return actions, err
		}
		actions = append(actions, a)
	}

	return actions, nil
}

// ensureGitRepo runs git init in dir unless dir is already in a git work tree,
// so the user can revert whatever the agent does next with git.
func ensureGitRepo(ctx context.Context, dir string) (fileAction, error) {
	if _, err := lookPath("git"); err != nil {
		return fileAction{Path: gitDir, Action: actionSkipped, Note: gitMissingNote}, nil
	}

	out, err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--is-inside-work-tree").Output()
	if err == nil && strings.TrimSpace(string(out)) == "true" {
		return fileAction{Path: gitDir, Action: actionUnchanged, Note: "already in a git work tree"}, nil
	}

	if out, err := exec.CommandContext(ctx, "git", "-C", dir, "init").CombinedOutput(); err != nil {
		return fileAction{}, fmt.Errorf("running git init in %s: %w: %s", dir, err, strings.TrimSpace(string(out)))
	}

	return fileAction{Path: gitDir, Action: actionCreated, Note: "git init"}, nil
}

func ensureSpecDir(dir string) (fileAction, error) {
	path := filepath.Join(dir, specDir)
	action := fileAction{Path: specDir + "/", Action: actionUnchanged}

	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		action.Action = actionCreated
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fileAction{}, fmt.Errorf("creating %s: %w", specDir, err)
	}

	return action, nil
}

// editFunc returns the content a file should have given its current content,
// which is nil when the file does not exist, and a note for the report when it
// changes the file.
type editFunc func(current []byte, exists bool) (next []byte, note string, err error)

// editFile writes dir/rel only when edit changes it, which is what makes a
// second run of init report every file as unchanged.
func editFile(dir, rel string, edit editFunc) (fileAction, error) {
	path := filepath.Join(dir, filepath.FromSlash(rel))

	current, err := os.ReadFile(path)
	exists := err == nil
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fileAction{}, fmt.Errorf("reading %s: %w", rel, err)
	}

	next, note, err := edit(current, exists)
	if err != nil {
		return fileAction{}, fmt.Errorf("updating %s: %w", rel, err)
	}

	if exists && bytes.Equal(current, next) {
		return fileAction{Path: rel, Action: actionUnchanged}, nil
	}

	if err := os.WriteFile(path, next, 0o644); err != nil {
		return fileAction{}, fmt.Errorf("writing %s: %w", rel, err)
	}

	if !exists {
		return fileAction{Path: rel, Action: actionCreated}, nil
	}
	return fileAction{Path: rel, Action: actionUpdated, Note: note}, nil
}

func writeIfMissing(content []byte) editFunc {
	return func(current []byte, exists bool) ([]byte, string, error) {
		if exists {
			return current, "", nil
		}
		return content, "", nil
	}
}

func addGitignoreLine(current []byte, _ bool) ([]byte, string, error) {
	for _, line := range strings.Split(string(current), "\n") {
		if strings.TrimSpace(line) == varFilePattern {
			return current, "", nil
		}
	}

	return appendLine(current, varFilePattern), "added " + varFilePattern, nil
}

type mcpServer struct {
	Type    string     `json:"type"`
	URL     string     `json:"url"`
	Headers mcpHeaders `json:"headers"`
}

type mcpHeaders struct {
	Authorization string `json:"Authorization"`
	WorkspaceID   string `json:"X-Workspace-ID"`
}

// mergeMCPServer sets mcpServers.rudderstack and leaves every other key as it
// was.
//
// The hosted MCP server's /bearer-auth-mcp route accepts a personal access
// token with X-Workspace-ID (verified live on 11 Oct 2026: initialize and
// list_sources both returned 200), so rudder-cli and MCP share one token.
// Writing .mcp.json before the agent starts is what makes MCP available in the
// agent's first session. The file holds the literal ${RUDDERSTACK_ACCESS_TOKEN},
// which Claude Code expands from the environment, never the token itself.
func mergeMCPServer(workspaceID string) editFunc {
	return func(current []byte, _ bool) ([]byte, string, error) {
		entry, err := json.Marshal(mcpServer{
			Type: "http",
			URL:  mcpServerURL,
			Headers: mcpHeaders{
				Authorization: "Bearer ${" + accessTokenEnv + "}",
				WorkspaceID:   workspaceID,
			},
		})
		if err != nil {
			return nil, "", err
		}

		doc := current
		if len(bytes.TrimSpace(doc)) == 0 {
			doc = []byte("{}")
		}
		if !gjson.ValidBytes(doc) || !gjson.ParseBytes(doc).IsObject() {
			return nil, "", errors.New("it is not a JSON object; fix or remove it and run init again")
		}

		servers := gjson.GetBytes(doc, "mcpServers")
		if servers.Exists() && !servers.IsObject() {
			return nil, "", errors.New("mcpServers is not a JSON object; fix or remove it and run init again")
		}

		note := "added the " + mcpServerName + " server"
		if existing := servers.Get(mcpServerName); existing.Exists() {
			same, err := sameJSON([]byte(existing.Raw), entry)
			if err != nil {
				return nil, "", err
			}
			if same {
				return current, "", nil
			}
			note = "replaced a different " + mcpServerName + " server entry"
		}

		next, err := sjson.SetRawBytes(doc, "mcpServers."+mcpServerName, entry)
		if err != nil {
			return nil, "", err
		}

		return pretty.Pretty(next), note, nil
	}
}

func sameJSON(a, b []byte) (bool, error) {
	var va, vb any
	if err := json.Unmarshal(a, &va); err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, &vb); err != nil {
		return false, err
	}
	return reflect.DeepEqual(va, vb), nil
}

func agentsBlock(workspaceID, workspaceName string) string {
	return agentsBlockOpen + "\n" + fmt.Sprintf(agentsBlockBody, workspaceName, workspaceID) + "\n" + agentsBlockEnd + "\n"
}

// upsertAgentsBlock replaces the text between the rudderstack markers, so the
// rest of a hand-written AGENTS.md survives every run.
func upsertAgentsBlock(block string) editFunc {
	return func(current []byte, exists bool) ([]byte, string, error) {
		text := string(current)

		begin := strings.Index(text, agentsBlockOpen)
		if begin == -1 {
			if !exists || len(bytes.TrimSpace(current)) == 0 {
				return []byte(block), "added the RudderStack block", nil
			}
			return []byte(string(appendLine(current, "")) + block), "added the RudderStack block", nil
		}

		end := strings.Index(text[begin:], agentsBlockEnd)
		if end == -1 {
			return nil, "", fmt.Errorf("found %q without a %q after it; fix or remove the marker and run init again", agentsBlockOpen, agentsBlockEnd)
		}
		end += begin + len(agentsBlockEnd)

		next := text[:begin] + strings.TrimSuffix(block, "\n") + text[end:]
		return []byte(next), "replaced the RudderStack block", nil
	}
}

// importAgents makes Claude Code, which reads CLAUDE.md and not AGENTS.md,
// load AGENTS.md through an @ import.
func importAgents(current []byte, exists bool) ([]byte, string, error) {
	if !exists {
		return []byte(agentsImport + "\n"), "", nil
	}
	if bytes.Contains(current, []byte(agentsFile)) {
		return current, "", nil
	}

	return appendLine(current, agentsImport), "added " + agentsImport, nil
}

// appendLine adds line to content on a line of its own.
func appendLine(content []byte, line string) []byte {
	out := string(content)
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return []byte(out + line + "\n")
}
