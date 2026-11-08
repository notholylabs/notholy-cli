// notholy: the NotHoly command line. Account, models, CVE data, file analysis, connectors for other agents,
// a local MCP bridge and an agent that runs in your terminal, on your machine, with your approval.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/notholylabs/notholy-cli/connectors"
	"github.com/notholylabs/notholy-cli/internal/api"
	"github.com/notholylabs/notholy-cli/internal/config"
)

const red = "\x1b[38;2;249;11;43m"
const dim = "\x1b[2m"
const reset = "\x1b[0m"

var stdout io.Writer = os.Stdout
var stderr io.Writer = os.Stderr

func color(s, c string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return c + s + reset
}

const usage = `NotHoly CLI

  notholy                       agent in this terminal
  notholy "question"            one answer (also reads stdin)
  notholy login | logout | whoami
  notholy models | credits
  notholy cve CVE-2024-3094
  notholy cve search "openssl" [--kev] [--min-cvss 7]
  notholy analyze ./app.apk     static analysis in the sandbox
  notholy connect <tool> [--global] [--undo]   claude, codex, opencode
  notholy run <tool> [args...]  start the tool on NotHoly for this process only
  notholy mcp                   MCP server on stdio
  notholy doctor
  notholy version

  --json on any lookup prints JSON.
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdin); err != nil {
		var e *api.Error
		if errors.As(err, &e) && e.Status == 401 {
			err = fmt.Errorf("%w (run notholy login)", err)
		}
		_, _ = fmt.Fprintln(stderr, color("✦ ", red)+err.Error())
		os.Exit(1)
	}
}

// split pulls --json out of the args, wherever it is.
func split(args []string) ([]string, bool) {
	out, js := []string{}, false
	for _, a := range args {
		if a == "--json" {
			js = true
			continue
		}
		out = append(out, a)
	}
	return out, js
}

func client() (*api.Client, config.Config, error) {
	cfg := config.Load()
	key, err := config.Key()
	if err != nil {
		return nil, cfg, err
	}
	c := api.New(cfg.APIBase, key)
	if cfg.Model == "" {
		cfg.Model, _ = c.DefaultModel(context.Background())
	}
	return c, cfg, nil
}

func printJSON(v any) error {
	b, _ := json.MarshalIndent(v, "", "  ")
	_, err := fmt.Fprintln(stdout, string(b))
	return err
}

func run(ctx context.Context, args []string, stdin io.Reader) error {
	args, js := split(args)
	if len(args) == 0 {
		if piped(stdin) {
			b, _ := io.ReadAll(io.LimitReader(stdin, 4<<20))
			return ask(ctx, string(b), js)
		}
		return agent(ctx, args, stdin)
	}
	switch args[0] {
	case "help", "-h", "--help":
		_, err := fmt.Fprint(stdout, usage)
		return err
	case "version", "--version":
		_, err := fmt.Fprintln(stdout, "notholy", api.Version)
		return err
	case "login":
		return login(ctx, args[1:])
	case "logout":
		config.DeleteKey()
		cfg := config.Load()
		cfg.OrgID, cfg.OrgName = "", ""
		_ = config.Save(cfg)
		_, err := fmt.Fprintln(stdout, "Signed out on this machine. Revoke the key in Connections if this device is lost.")
		return err
	case "token": // used by Claude Code's apiKeyHelper; prints the key and nothing else
		k, err := config.Key()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, k)
		return err
	case "whoami":
		return whoami(ctx, js)
	case "models":
		return models(ctx, js)
	case "credits":
		return credits(ctx, js)
	case "cve":
		return cve(ctx, args[1:], js)
	case "analyze":
		return analyze(ctx, args[1:], js)
	case "connect":
		return connect(args[1:])
	case "run":
		return runTool(args[1:])
	case "mcp":
		return mcpBridge(ctx, stdin, stdout)
	case "doctor":
		return doctor(ctx)
	case "agent":
		return agent(ctx, args[1:], stdin)
	}
	if strings.HasPrefix(args[0], "-") {
		return agent(ctx, args, stdin)
	}
	q := strings.Join(args, " ")
	if piped(stdin) {
		b, _ := io.ReadAll(io.LimitReader(stdin, 4<<20))
		q += "\n\n" + string(b)
	}
	return ask(ctx, q, js)
}

func piped(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return r != nil
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice == 0
}

// ---------- account and lookups ----------

func whoami(ctx context.Context, js bool) error {
	c, cfg, err := client()
	if err != nil {
		return err
	}
	var cr map[string]any
	if err := c.JSON(ctx, "GET", "/credits", nil, &cr); err != nil {
		return err
	}
	info := map[string]any{"organization": cfg.OrgName, "organization_id": cfg.OrgID, "key": config.Mask(c.Key), "api_base": cfg.APIBase, "credits": cr}
	if js {
		return printJSON(info)
	}
	_, err = fmt.Fprintf(stdout, "%s %s\n  key     %s\n  api     %s\n  balance %v\n", color("✦", red), orDash(cfg.OrgName), config.Mask(c.Key), cfg.APIBase, money(cr))
	return err
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func money(cr map[string]any) string {
	for _, k := range []string{"available", "balance"} {
		if v, ok := cr[k]; ok {
			return fmt.Sprintf("$%v", v)
		}
	}
	b, _ := json.Marshal(cr)
	return string(b)
}

func credits(ctx context.Context, js bool) error {
	c, _, err := client()
	if err != nil {
		return err
	}
	var cr map[string]any
	if err := c.JSON(ctx, "GET", "/credits", nil, &cr); err != nil {
		return err
	}
	if js {
		return printJSON(cr)
	}
	_, err = fmt.Fprintln(stdout, money(cr))
	return err
}

func models(ctx context.Context, js bool) error {
	c, _, err := client()
	if err != nil {
		return err
	}
	var r struct {
		Data []map[string]any `json:"data"`
	}
	if err := c.JSON(ctx, "GET", "/models", nil, &r); err != nil {
		return err
	}
	if js {
		return printJSON(r.Data)
	}
	for _, m := range r.Data {
		price := ""
		if p, ok := m["pricing"].(map[string]any); ok {
			price = fmt.Sprintf("$%v in / $%v out per 1M", p["input"], p["output"])
		}
		_, _ = fmt.Fprintf(stdout, "%-18v %-10v %s\n", m["id"], ctxLabel(m["context_length"]), price)
	}
	return nil
}

func ctxLabel(v any) string {
	if n, ok := v.(float64); ok && n > 0 {
		return fmt.Sprintf("%.0fK ctx", n/1000)
	}
	return ""
}

func cve(ctx context.Context, args []string, js bool) error {
	if len(args) == 0 {
		return errors.New("usage: notholy cve CVE-2024-3094 | notholy cve search <text> [--kev] [--min-cvss N]")
	}
	c, _, err := client()
	if err != nil {
		return err
	}
	if args[0] == "search" {
		fs := flag.NewFlagSet("cve search", flag.ContinueOnError)
		kev := fs.Bool("kev", false, "only CISA KEV")
		minCvss := fs.Float64("min-cvss", 0, "minimum CVSS")
		limit := fs.Int("limit", 10, "results")
		rest := reorder(args[1:])
		if err := fs.Parse(rest); err != nil {
			return err
		}
		q := url.Values{"q": {strings.Join(fs.Args(), " ")}, "limit": {fmt.Sprint(*limit)}}
		if *kev {
			q.Set("kev", "true")
		}
		if *minCvss > 0 {
			q.Set("min_cvss", fmt.Sprint(*minCvss))
		}
		var r struct {
			Data []map[string]any `json:"data"`
		}
		if err := c.JSON(ctx, "GET", "/cve/search?"+q.Encode(), nil, &r); err != nil {
			return err
		}
		if js {
			return printJSON(r.Data)
		}
		for _, x := range r.Data {
			_, _ = fmt.Fprintf(stdout, "%-16v cvss %-4v %v\n", x["id"], x["cvss"], trim(fmt.Sprint(x["description"]), 90))
		}
		return nil
	}
	var rec map[string]any
	if err := c.JSON(ctx, "GET", "/cve/"+url.PathEscape(strings.ToUpper(args[0])), nil, &rec); err != nil {
		return err
	}
	if js {
		return printJSON(rec)
	}
	_, err = fmt.Fprintf(stdout, "%s %v\n  cvss %v · epss %v · kev %v\n  %v\n", color("✦", red), rec["id"], rec["cvss"], rec["epss"], rec["kev"], rec["description"])
	return err
}

// reorder moves flags before positional args, so `search openssl --kev` works with the flag package.
func reorder(args []string) []string {
	flags, pos := []string{}, []string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--") {
			flags = append(flags, a)
			if !strings.Contains(a, "=") && a != "--kev" && i+1 < len(args) {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		pos = append(pos, a)
	}
	return append(flags, pos...)
}

func trim(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

func analyze(ctx context.Context, args []string, js bool) error {
	if len(args) != 1 {
		return errors.New("usage: notholy analyze <file>")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	c, _, err := client()
	if err != nil {
		return err
	}
	var info map[string]any
	if err := c.Upload(ctx, filepath.Base(args[0]), data, &info); err != nil {
		return err
	}
	if js {
		return printJSON(info)
	}
	_, err = fmt.Fprintf(stdout, "%s %v · %v · %v\n  Kept for 1 hour. Ask about it: notholy \"summarize file %v\"\n", color("✦", red), info["name"], info["status"], info["kind"], info["id"])
	return err
}

// ---------- connectors ----------

func connCtx() (connectors.Ctx, error) {
	cfg := config.Load()
	home, _ := os.UserHomeDir()
	if h := os.Getenv("NOTHOLY_CONNECT_HOME"); h != "" {
		home = h
	}
	wd, _ := os.Getwd()
	if cfg.Model == "" {
		m, err := api.New(cfg.APIBase, "").DefaultModel(context.Background())
		if err != nil {
			return connectors.Ctx{}, fmt.Errorf("could not read the model catalog: %w", err)
		}
		cfg.Model = m
	}
	return connectors.Ctx{APIBase: cfg.APIBase, Model: cfg.Model, Home: home, Project: wd}, nil
}

func connect(args []string) error {
	fs := flag.NewFlagSet("connect", flag.ContinueOnError)
	global := fs.Bool("global", false, "write the user-wide config")
	undo := fs.Bool("undo", false, "restore the files from before connect")
	if err := fs.Parse(reorder(args)); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: notholy connect <%s> [--global] [--undo]", strings.Join(connectors.Names(), "|"))
	}
	c, ok := connectors.Get(fs.Arg(0))
	if !ok {
		return fmt.Errorf("unknown tool %q; try %s", fs.Arg(0), strings.Join(connectors.Names(), ", "))
	}
	ctx, _ := connCtx()
	if *undo {
		files, err := c.Undo(ctx, *global)
		for _, f := range files {
			_, _ = fmt.Fprintln(stdout, "restored", f)
		}
		return err
	}
	files, err := c.Connect(ctx, *global)
	if err != nil {
		return err
	}
	for _, f := range files {
		_, _ = fmt.Fprintln(stdout, color("✦", red), "wrote", f, dim+"(backup kept; undo with --undo)"+reset)
	}
	switch c.Name {
	case "codex":
		_, _ = fmt.Fprintln(stdout, "  Start it with: codex --profile notholy   (needs NOTHOLY_API_KEY in your shell, or use notholy run codex)")
	case "opencode":
		_, _ = fmt.Fprintln(stdout, "  OpenCode reads the key from NOTHOLY_API_KEY, or use notholy run opencode")
	}
	return nil
}

func runTool(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: notholy run <%s> [args...]", strings.Join(connectors.Names(), "|"))
	}
	c, ok := connectors.Get(args[0])
	if !ok {
		return fmt.Errorf("unknown tool %q; try %s", args[0], strings.Join(connectors.Names(), ", "))
	}
	key, err := config.Key()
	if err != nil {
		return err
	}
	ctx, _ := connCtx()
	ctx.Key = key
	cmd, err := c.Command(ctx, args[1:])
	if err != nil {
		return err
	}
	signal.Ignore(os.Interrupt) // the child handles Ctrl+C
	if err := cmd.Run(); err != nil {
		var ee interface{ ExitCode() int }
		if errors.As(err, &ee) {
			os.Exit(ee.ExitCode())
		}
		return err
	}
	return nil
}

// ---------- doctor ----------

func doctor(ctx context.Context) error {
	cfg := config.Load()
	check := func(name string, f func() error) {
		t := time.Now()
		if err := f(); err != nil {
			_, _ = fmt.Fprintf(stdout, "  ✗ %-10s %v\n", name, err)
			return
		}
		_, _ = fmt.Fprintf(stdout, "  %s %-10s %s\n", color("✦", red), name, time.Since(t).Round(time.Millisecond))
	}
	_, _ = fmt.Fprintln(stdout, "api", cfg.APIBase)
	c, _, err := client()
	check("account", func() error {
		if err != nil {
			return err
		}
		var cr map[string]any
		return c.JSON(ctx, "GET", "/credits", nil, &cr)
	})
	if err != nil {
		return nil
	}
	check("models", func() error {
		var r map[string]any
		return c.JSON(ctx, "GET", "/models", nil, &r)
	})
	check("streaming", func() error {
		got := ""
		m := cfg.Model
		if m == "" {
			var err error
			if m, err = c.DefaultModel(ctx); err != nil {
				return err
			}
		}
		_, err := c.Stream(ctx, api.ChatRequest{Model: m, Messages: []api.Message{{Role: "user", Content: "Reply with: ok"}}}, func(s string) { got += s }, nil)
		if err == nil && got == "" {
			return errors.New("no text came back")
		}
		return err
	})
	for _, name := range connectors.Names() {
		cn, _ := connectors.Get(name)
		check(name, func() error {
			if _, err := cn.Command(connectors.Ctx{APIBase: cfg.APIBase, Model: cfg.Model}, nil); err != nil {
				return err
			}
			return nil
		})
	}
	return nil
}
