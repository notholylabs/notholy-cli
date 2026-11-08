package connectors

import "path/filepath"

// Claude Code reads the Anthropic variables. Model ids without "claude" are not auto-discovered, so they are set
// explicitly. `connect` stores no secret: Claude Code asks `notholy token` for the key through apiKeyHelper.
func claudeEnv(c Ctx) map[string]string {
	big := c.Model + "[1m]"
	return map[string]string{
		"ANTHROPIC_BASE_URL":                     c.Origin(),
		"ANTHROPIC_MODEL":                        big,
		"ANTHROPIC_DEFAULT_OPUS_MODEL":           big,
		"ANTHROPIC_DEFAULT_SONNET_MODEL":         big,
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":          "notholy-pulse",
		"CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS": "1",
		"CLAUDE_CODE_ATTRIBUTION_HEADER":         "0",
	}
}

func init() {
	register(&Connector{
		Name:    "claude",
		Aliases: []string{"claude-code"},
		Bin:     "claude",
		Env: func(c Ctx) map[string]string {
			e := claudeEnv(c)
			e["ANTHROPIC_AUTH_TOKEN"] = c.Key
			e["ANTHROPIC_API_KEY"] = "" // a stray Anthropic key in the shell would win otherwise
			return e
		},
		Files: func(c Ctx, global bool) []string {
			if global {
				return []string{filepath.Join(c.Home, ".claude", "settings.json")}
			}
			return []string{filepath.Join(c.Project, ".claude", "settings.local.json")}
		},
		Write: func(c Ctx, global bool) error {
			path := filepath.Join(c.Project, ".claude", "settings.local.json")
			if global {
				path = filepath.Join(c.Home, ".claude", "settings.json")
			}
			m, err := readJSON(path)
			if err != nil {
				return err
			}
			env := obj(m, "env")
			for k, v := range claudeEnv(c) {
				env[k] = v
			}
			m["apiKeyHelper"] = "notholy token"
			return writeJSON(path, m)
		},
	})
}
