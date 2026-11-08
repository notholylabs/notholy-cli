package main

// The terminal agent. It runs on the user's machine and only acts with the user's approval, by mode:
//   plan       reads only
//   ask        default; asks before every write and command
//   auto-edit  edits files without asking; commands still ask
//   yolo       asks nothing; needs --yolo on every start
// File contents and command output go back to the model as data, never as instructions.
// ponytail: line-based interface; a Bubble Tea view can replace the printing later without touching the loop.

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/notholylabs/notholy-cli/internal/api"
	"github.com/notholylabs/notholy-cli/internal/config"
)

type session struct {
	ID       string        `json:"id"`
	Model    string        `json:"model"`
	Effort   string        `json:"effort,omitempty"`
	Mode     string        `json:"mode"`
	Messages []api.Message `json:"messages"`
	In       int           `json:"input_tokens"`
	Out      int           `json:"output_tokens"`
}

const maxOutput = 30_000 // characters of tool output sent back per call

// systemPrompt is only where the model runs, like the API sends nothing of ours (owner decision 2026-11-07);
// NOTHOLY.md from the project, when present, follows as the user wrote it.
func systemPrompt(project string) string {
	p := "Running on NotHoly.com."
	if project != "" {
		p += "\n\n" + project
	}
	return p
}

func fn(name, desc string, props map[string]any, req ...string) api.Tool {
	t := api.Tool{Type: "function"}
	t.Function.Name, t.Function.Description = name, desc
	t.Function.Parameters = map[string]any{"type": "object", "properties": props, "required": req}
	return t
}

var s = func(d string) map[string]any { return map[string]any{"type": "string", "description": d} }

var agentTools = []api.Tool{
	fn("read", "Read a text file. Optional offset and limit are line numbers.", map[string]any{"path": s("file path"), "offset": map[string]any{"type": "integer"}, "limit": map[string]any{"type": "integer"}}, "path"),
	fn("grep", "Search file contents with a regular expression under a directory.", map[string]any{"pattern": s("RE2 regular expression"), "path": s("directory, default .")}, "pattern"),
	fn("glob", "List files matching a glob such as **/*.go.", map[string]any{"pattern": s("glob")}, "pattern"),
	fn("edit", "Replace one exact occurrence of old with new in a file.", map[string]any{"path": s("file"), "old": s("exact text to replace"), "new": s("replacement")}, "path", "old", "new"),
	fn("write", "Create or overwrite a file.", map[string]any{"path": s("file"), "content": s("full content")}, "path", "content"),
	fn("bash", "Run a shell command in the working directory and return its output.", map[string]any{"command": s("command line")}, "command"),
	fn("cve_lookup", "Consolidated CVE record from NotHoly: CVSS, EPSS, KEV, references.", map[string]any{"id": s("CVE id")}, "id"),
	fn("web_search", "Search the public web through NotHoly.", map[string]any{"query": s("query")}, "query"),
}

var writeTools = map[string]bool{"edit": true, "write": true}

func agent(ctx context.Context, args []string, stdin io.Reader) error {
	f := flag.NewFlagSet("agent", flag.ContinueOnError)
	mode := f.String("mode", "ask", "plan, ask, auto-edit or yolo")
	yolo := f.Bool("yolo", false, "allow yolo mode for this run")
	model := f.String("model", "", "model id")
	prompt := f.String("p", "", "run one prompt and exit")
	if err := f.Parse(args); err != nil {
		return err
	}
	if *mode == "yolo" && !*yolo {
		return errors.New("yolo mode runs commands without asking; start it with --mode yolo --yolo")
	}
	if *yolo {
		*mode = "yolo"
	}
	switch *mode {
	case "plan", "ask", "auto-edit", "yolo":
	default:
		return fmt.Errorf("unknown mode %q", *mode)
	}
	c, cfg, err := client()
	if err != nil {
		return err
	}
	in := bufio.NewReader(stdin)
	project, _ := os.ReadFile("NOTHOLY.md")
	ss := &session{ID: time.Now().UTC().Format("20060102-150405"), Model: cfg.Model, Mode: *mode}
	if *model != "" {
		ss.Model = *model
	}
	ss.Messages = []api.Message{{Role: "system", Content: systemPrompt(trim(string(project), 20000))}}

	if *prompt != "" {
		return turn(ctx, c, ss, *prompt, in)
	}
	_, _ = fmt.Fprintf(stdout, "%s NotHoly · %s · mode %s · /help\n", color("✦", red), ss.Model, ss.Mode)
	for {
		_, _ = fmt.Fprint(stdout, color("\n› ", red))
		line, err := in.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" && err != nil {
			return nil
		}
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "/") {
			if done := slash(ctx, c, ss, line); done {
				return nil
			}
			continue
		}
		if err := turn(ctx, c, ss, line, in); err != nil {
			_, _ = fmt.Fprintln(stderr, color("✦ ", red)+err.Error())
		}
	}
}

func slash(ctx context.Context, c *api.Client, ss *session, line string) (exit bool) {
	f := strings.Fields(line)
	arg := ""
	if len(f) > 1 {
		arg = f[1]
	}
	switch f[0] {
	case "/exit", "/quit":
		return true
	case "/model":
		if arg != "" {
			ss.Model = arg
		}
		_, _ = fmt.Fprintln(stdout, "model", ss.Model)
	case "/effort":
		// the gateway maps each level to the nearest one the model supports (catalog reasoning_efforts)
		switch arg {
		case "off", "low", "high", "max":
			ss.Effort = arg
		}
		_, _ = fmt.Fprintln(stdout, "effort", orDash(ss.Effort))
	case "/mode":
		if arg == "plan" || arg == "ask" || arg == "auto-edit" {
			ss.Mode = arg
		}
		_, _ = fmt.Fprintln(stdout, "mode", ss.Mode, dim+"(yolo only with --yolo at start)"+reset)
	case "/cost":
		var cr map[string]any
		_ = c.JSON(ctx, "GET", "/credits", nil, &cr)
		_, _ = fmt.Fprintf(stdout, "this session: %d input + %d output tokens · balance %s\n", ss.In, ss.Out, money(cr))
	case "/compact":
		if len(ss.Messages) > 7 {
			ss.Messages = append(ss.Messages[:1], ss.Messages[len(ss.Messages)-6:]...)
		}
		_, _ = fmt.Fprintln(stdout, "kept the instructions and the last 6 messages")
	case "/resume":
		if prev, err := latestSession(); err == nil {
			*ss = *prev
			_, _ = fmt.Fprintf(stdout, "resumed %s (%d messages)\n", ss.ID, len(ss.Messages))
		} else {
			_, _ = fmt.Fprintln(stdout, "no saved session")
		}
	case "/mcp":
		_, _ = fmt.Fprintln(stdout, "Add NotHoly to another agent: claude mcp add notholy -- notholy mcp")
	default:
		_, _ = fmt.Fprintln(stdout, "/model <id>  /effort off|low|high|max  /mode plan|ask|auto-edit  /cost  /compact  /resume  /mcp  /exit")
	}
	return false
}

// turn sends the user's message and keeps going while the model calls tools.
func turn(ctx context.Context, c *api.Client, ss *session, text string, in *bufio.Reader) error {
	ss.Messages = append(ss.Messages, api.Message{Role: "user", Content: text})
	defer saveSession(ss)
	for step := 0; step < 40; step++ {
		thinking := false
		res, err := c.Stream(ctx, api.ChatRequest{Model: ss.Model, Messages: ss.Messages, Tools: agentTools, ReasoningEffort: ss.Effort},
			func(t string) {
				if thinking {
					_, _ = fmt.Fprint(stdout, "\n")
					thinking = false
				}
				_, _ = fmt.Fprint(stdout, t)
			},
			func(r string) {
				if !thinking {
					_, _ = fmt.Fprint(stdout, color("✦ thinking ", red))
					thinking = true
				}
				_, _ = fmt.Fprint(stdout, dim+r+reset)
			})
		if err != nil {
			return err
		}
		ss.In += res.Usage.PromptTokens
		ss.Out += res.Usage.CompletionTokens
		ss.Messages = append(ss.Messages, res.Message)
		if len(res.Message.ToolCalls) == 0 {
			_, _ = fmt.Fprintln(stdout)
			return nil
		}
		for _, tc := range res.Message.ToolCalls {
			out := callTool(ctx, c, ss.Mode, tc, in)
			ss.Messages = append(ss.Messages, api.Message{Role: "tool", ToolCallID: tc.ID, Content: "<tool_output>\n" + out + "\n</tool_output>"})
		}
	}
	return errors.New("stopped after 40 tool steps")
}

func approve(mode, tool, summary string, in *bufio.Reader) bool {
	needs := map[string]bool{"plan": false, "ask": true, "auto-edit": !writeTools[tool], "yolo": false}[mode]
	if mode == "plan" {
		return false
	}
	if !needs {
		return true
	}
	_, _ = fmt.Fprintf(stdout, "\n%s %s %s\n  allow? [y/N] ", color("✦", red), tool, summary)
	ans, _ := in.ReadString('\n')
	ans = strings.ToLower(strings.TrimSpace(ans))
	return ans == "y" || ans == "yes"
}

func callTool(ctx context.Context, c *api.Client, mode string, tc api.ToolCall, in *bufio.Reader) string {
	var a map[string]any
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &a); err != nil {
		return "invalid arguments: " + err.Error()
	}
	str := func(k string) string { v, _ := a[k].(string); return v }
	name := tc.Function.Name
	_, _ = fmt.Fprintf(stdout, "\n%s %s %s\n", color("✦", red), name, dim+trim(tc.Function.Arguments, 120)+reset)
	switch name {
	case "read":
		b, err := os.ReadFile(str("path"))
		if err != nil {
			return err.Error()
		}
		lines := strings.Split(string(b), "\n")
		off, _ := a["offset"].(float64)
		lim, _ := a["limit"].(float64)
		start := min(max(int(off), 0), len(lines))
		end := len(lines)
		if lim > 0 {
			end = min(start+int(lim), len(lines))
		}
		return clip(strings.Join(lines[start:end], "\n"))
	case "glob":
		return clip(strings.Join(globAll(str("pattern")), "\n"))
	case "grep":
		re, err := regexp.Compile(str("pattern"))
		if err != nil {
			return err.Error()
		}
		root := str("path")
		if root == "" {
			root = "."
		}
		return clip(grepDir(root, re))
	case "edit", "write", "bash":
		summary := str("path")
		if name == "bash" {
			summary = str("command")
		}
		if !approve(mode, name, summary, in) {
			if mode == "plan" {
				return "refused: plan mode only reads"
			}
			return "the user refused this action"
		}
		switch name {
		case "write":
			if err := os.MkdirAll(filepath.Dir(str("path")), 0o755); err != nil {
				return err.Error()
			}
			if err := os.WriteFile(str("path"), []byte(str("content")), 0o644); err != nil {
				return err.Error()
			}
			return "wrote " + str("path")
		case "edit":
			b, err := os.ReadFile(str("path"))
			if err != nil {
				return err.Error()
			}
			if n := strings.Count(string(b), str("old")); n != 1 {
				return fmt.Sprintf("old text found %d times; it must match exactly once", n)
			}
			if err := os.WriteFile(str("path"), []byte(strings.Replace(string(b), str("old"), str("new"), 1)), 0o644); err != nil {
				return err.Error()
			}
			return "edited " + str("path")
		default:
			return runShell(ctx, str("command"))
		}
	case "cve_lookup":
		var rec map[string]any
		if err := c.JSON(ctx, "GET", "/cve/"+strings.ToUpper(str("id")), nil, &rec); err != nil {
			return err.Error()
		}
		b, _ := json.Marshal(rec)
		return clip(string(b))
	case "web_search":
		var r map[string]any
		if err := c.JSON(ctx, "POST", "/web/search", map[string]any{"query": str("query")}, &r); err != nil {
			return err.Error()
		}
		b, _ := json.Marshal(r)
		return clip(string(b))
	}
	return "unknown tool " + name
}

func runShell(ctx context.Context, command string) string {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	out, err := cmd.CombinedOutput()
	res := string(out)
	if err != nil {
		res += "\n[exit: " + err.Error() + "]"
	}
	_, _ = fmt.Fprint(stdout, dim+trim(res, 400)+reset+"\n")
	return clip(res)
}

func clip(s string) string {
	if len(s) > maxOutput {
		return s[:maxOutput] + "\n[cut at 30,000 characters]"
	}
	return s
}

func globAll(pattern string) []string {
	out := []string{}
	if !strings.Contains(pattern, "**") {
		m, _ := filepath.Glob(pattern)
		return m
	}
	suffix := strings.TrimPrefix(pattern[strings.Index(pattern, "**")+2:], "/")
	_ = filepath.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if ok, _ := filepath.Match(suffix, d.Name()); ok && !d.IsDir() {
			out = append(out, p)
		}
		if len(out) >= 500 {
			return fs.SkipAll
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func grepDir(root string, re *regexp.Regexp) string {
	var b strings.Builder
	n := 0
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil || len(data) > 2<<20 || strings.ContainsRune(string(data[:min(len(data), 512)]), 0) {
			return nil
		}
		for i, line := range strings.Split(string(data), "\n") {
			if re.MatchString(line) {
				fmt.Fprintf(&b, "%s:%d:%s\n", p, i+1, trim(line, 200))
				if n++; n >= 200 {
					return fs.SkipAll
				}
			}
		}
		return nil
	})
	return b.String()
}

// ---------- sessions (~/.notholy/sessions, mode 600) ----------

func sessionDir() string { return filepath.Join(config.Dir(), "sessions") }

func saveSession(ss *session) {
	if err := os.MkdirAll(sessionDir(), 0o700); err != nil {
		return
	}
	b, _ := json.Marshal(ss)
	_ = os.WriteFile(filepath.Join(sessionDir(), ss.ID+".json"), b, 0o600)
}

func latestSession() (*session, error) {
	m, _ := filepath.Glob(filepath.Join(sessionDir(), "*.json"))
	if len(m) == 0 {
		return nil, os.ErrNotExist
	}
	sort.Strings(m)
	b, err := os.ReadFile(m[len(m)-1])
	if err != nil {
		return nil, err
	}
	var ss session
	return &ss, json.Unmarshal(b, &ss)
}

// ask is `notholy "question"`: one streamed answer, no tools.
func ask(ctx context.Context, q string, js bool) error {
	c, cfg, err := client()
	if err != nil {
		return err
	}
	onText := func(t string) { _, _ = fmt.Fprint(stdout, t) }
	if js {
		onText = nil
	}
	res, err := c.Stream(ctx, api.ChatRequest{Model: cfg.Model, Messages: []api.Message{{Role: "system", Content: systemPrompt("")}, {Role: "user", Content: q}}}, onText, nil)
	if err != nil {
		return err
	}
	if js {
		return printJSON(map[string]any{"model": cfg.Model, "answer": res.Message.Content, "usage": res.Usage})
	}
	_, err = fmt.Fprintln(stdout)
	return err
}
