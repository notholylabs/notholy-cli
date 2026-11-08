package connectors

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ctx(t *testing.T) Ctx {
	d := t.TempDir()
	return Ctx{APIBase: "http://127.0.0.1:9/v1", Key: "nh_cli_test", Model: "notholy-specter", Home: filepath.Join(d, "home"), Project: filepath.Join(d, "proj")}
}

func TestClaudeConnectMergesAndUndoRestores(t *testing.T) {
	c := ctx(t)
	path := filepath.Join(c.Project, ".claude", "settings.local.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	orig := `{"permissions":{"allow":["Bash(ls:*)"]},"env":{"KEEP":"1"}}`
	if err := os.WriteFile(path, []byte(orig), 0o600); err != nil {
		t.Fatal(err)
	}
	cn, _ := Get("claude-code")
	if _, err := cn.Connect(c, false); err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	b, _ := os.ReadFile(path)
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	env := m["env"].(map[string]any)
	if env["KEEP"] != "1" || env["ANTHROPIC_BASE_URL"] != "http://127.0.0.1:9" || env["ANTHROPIC_MODEL"] != "notholy-specter[1m]" {
		t.Fatalf("env not merged: %v", env)
	}
	if m["apiKeyHelper"] != "notholy token" || strings.Contains(string(b), "nh_cli_test") {
		t.Fatalf("the key must never be written: %s", b)
	}
	if m["permissions"] == nil {
		t.Fatal("user settings were dropped")
	}
	// a second connect keeps the first backup
	if _, err := cn.Connect(c, false); err != nil {
		t.Fatal(err)
	}
	if _, err := cn.Undo(c, false); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(path)
	if string(b) != orig {
		t.Fatalf("undo did not restore: %s", b)
	}
	if _, err := cn.Undo(c, false); err == nil {
		t.Fatal("second undo should say there is nothing to undo")
	}
}

func TestCodexBlockIsReplacedNotDuplicatedAndUndoRemovesNewFile(t *testing.T) {
	c := ctx(t)
	cn, _ := Get("codex")
	for range 2 {
		if _, err := cn.Connect(c, false); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(c.Home, ".codex", "config.toml")
	b, _ := os.ReadFile(path)
	if strings.Count(string(b), "[model_providers.notholy]") != 1 || !strings.Contains(string(b), `wire_api = "responses"`) {
		t.Fatalf("bad block:\n%s", b)
	}
	if _, err := cn.Undo(c, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("config.toml did not exist before connect and should be gone after undo")
	}
}

func TestOpencodeKeepsUserModelAndRunEnvCarriesConfig(t *testing.T) {
	c := ctx(t)
	path := filepath.Join(c.Home, ".config", "opencode", "opencode.json")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(`{"model":"anthropic/x","theme":"dark"}`), 0o600)
	cn, _ := Get("opencode")
	if _, err := cn.Connect(c, false); err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	b, _ := os.ReadFile(path)
	_ = json.Unmarshal(b, &m)
	if m["model"] != "anthropic/x" || m["theme"] != "dark" || m["provider"].(map[string]any)["notholy"] == nil {
		t.Fatalf("merge wrong: %s", b)
	}
	env := cn.Env(c)
	if env["NOTHOLY_API_KEY"] != "nh_cli_test" || !strings.Contains(env["OPENCODE_CONFIG_CONTENT"], `"baseURL":"http://127.0.0.1:9/v1"`) {
		t.Fatalf("run env: %v", env)
	}
	if strings.Contains(env["OPENCODE_CONFIG_CONTENT"], "nh_cli_test") {
		t.Fatal("the config content must reference the env var, not the key")
	}
}

func TestInvalidJSONIsLeftAlone(t *testing.T) {
	c := ctx(t)
	path := filepath.Join(c.Home, ".config", "opencode", "opencode.json")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(`{ not json`), 0o600)
	cn, _ := Get("opencode")
	if _, err := cn.Connect(c, false); err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
	b, _ := os.ReadFile(path)
	if string(b) != `{ not json` {
		t.Fatal("file was changed")
	}
}
