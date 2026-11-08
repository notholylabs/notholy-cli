// Package connectors points third-party agents at NotHoly.
//
// `notholy run <tool>` sets everything in the child process only and changes no file. `notholy connect <tool>`
// writes the tool's config, backs up the file first and `--undo` puts it back. Each tool has its own file here.
package connectors

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Ctx is what a connector needs to know about the account.
type Ctx struct {
	APIBase string // https://api.notholy.com/v1
	Key     string
	Model   string
	Home    string // user home, overridable in tests
	Project string // working directory, for project-level config
}

// Origin is the API base without /v1, for clients that add the path themselves.
func (c Ctx) Origin() string { return strings.TrimSuffix(strings.TrimRight(c.APIBase, "/"), "/v1") }

type Connector struct {
	Name    string
	Aliases []string
	Bin     string
	// Env and Args are applied to `notholy run` only.
	Env  func(Ctx) map[string]string
	Args func(Ctx) []string
	// Files written by `connect`, and what to put in them.
	Files func(c Ctx, global bool) []string
	Write func(c Ctx, global bool) error
}

var all = map[string]*Connector{}

func register(c *Connector) {
	all[c.Name] = c
	for _, a := range c.Aliases {
		all[a] = c
	}
}

func Get(name string) (*Connector, bool) {
	c, ok := all[strings.ToLower(name)]
	return c, ok
}

func Names() []string {
	seen := map[string]bool{}
	out := []string{}
	for _, c := range all {
		if !seen[c.Name] {
			seen[c.Name] = true
			out = append(out, c.Name)
		}
	}
	sort.Strings(out)
	return out
}

// Command builds the child process for `notholy run`: the tool, its args, and our env on top of the user's.
func (c *Connector) Command(ctx Ctx, extra []string) (*exec.Cmd, error) {
	bin, err := exec.LookPath(c.Bin)
	if err != nil {
		return nil, fmt.Errorf("%s is not installed or not on PATH", c.Bin)
	}
	args := append([]string{}, extra...)
	if c.Args != nil {
		args = append(c.Args(ctx), extra...)
	}
	cmd := exec.Command(bin, args...)
	env := os.Environ()
	for k, v := range c.Env(ctx) {
		env = append(env, k+"="+v)
	}
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd, nil
}

// ---------- backups ----------

const bakSuffix = ".notholy-backup"
const noneSuffix = ".notholy-absent"

// backup keeps the original once: a copy, or a marker saying the file did not exist.
func backup(path string) error {
	if _, err := os.Stat(path + bakSuffix); err == nil {
		return nil
	}
	if _, err := os.Stat(path + noneSuffix); err == nil {
		return nil
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path+noneSuffix, nil, 0o600)
	}
	if err != nil {
		return err
	}
	return os.WriteFile(path+bakSuffix, b, 0o600)
}

// Connect backs up every file the tool uses, then writes the config.
func (c *Connector) Connect(ctx Ctx, global bool) ([]string, error) {
	files := c.Files(ctx, global)
	for _, f := range files {
		if err := backup(f); err != nil {
			return nil, err
		}
	}
	return files, c.Write(ctx, global)
}

// Undo restores each file from its backup, or removes it if it did not exist before.
func (c *Connector) Undo(ctx Ctx, global bool) ([]string, error) {
	restored := []string{}
	for _, f := range c.Files(ctx, global) {
		if b, err := os.ReadFile(f + bakSuffix); err == nil {
			if err := os.WriteFile(f, b, 0o600); err != nil {
				return restored, err
			}
			_ = os.Remove(f + bakSuffix)
			restored = append(restored, f)
			continue
		}
		if _, err := os.Stat(f + noneSuffix); err == nil {
			_ = os.Remove(f)
			_ = os.Remove(f + noneSuffix)
			restored = append(restored, f)
		}
	}
	if len(restored) == 0 {
		return nil, fmt.Errorf("nothing to undo for %s", c.Name)
	}
	return restored, nil
}

// ---------- JSON helpers ----------

func readJSON(path string) (map[string]any, error) {
	m := map[string]any{}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) || (err == nil && len(strings.TrimSpace(string(b))) == 0) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s is not valid JSON, so it was left alone: %w", path, err)
	}
	return m, nil
}

func writeJSON(path string, m map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

func obj(m map[string]any, k string) map[string]any {
	if v, ok := m[k].(map[string]any); ok {
		return v
	}
	v := map[string]any{}
	m[k] = v
	return v
}
