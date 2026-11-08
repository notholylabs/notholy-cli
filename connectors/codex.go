package connectors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Codex speaks the Responses API. `run` passes the provider as -c overrides; `connect` adds a provider and a
// `notholy` profile to ~/.codex/config.toml between markers, leaving the user's own top-level keys alone.
const codexBegin = "# >>> notholy (managed by `notholy connect codex`)"
const codexEnd = "# <<< notholy"

func codexBlock(c Ctx) string {
	return fmt.Sprintf(`%s
[model_providers.notholy]
name = "NotHoly"
base_url = %q
env_key = "NOTHOLY_API_KEY"
wire_api = "responses"

[profiles.notholy]
model = %q
model_provider = "notholy"
%s
`, codexBegin, c.APIBase, c.Model, codexEnd)
}

func init() {
	register(&Connector{
		Name: "codex",
		Bin:  "codex",
		Env:  func(c Ctx) map[string]string { return map[string]string{"NOTHOLY_API_KEY": c.Key} },
		Args: func(c Ctx) []string {
			return []string{
				"-c", `model_providers.notholy.name="NotHoly"`,
				"-c", fmt.Sprintf("model_providers.notholy.base_url=%q", c.APIBase),
				"-c", `model_providers.notholy.env_key="NOTHOLY_API_KEY"`,
				"-c", `model_providers.notholy.wire_api="responses"`,
				"-c", `model_provider="notholy"`,
				"-c", fmt.Sprintf("model=%q", c.Model),
			}
		},
		Files: func(c Ctx, _ bool) []string { return []string{filepath.Join(c.Home, ".codex", "config.toml")} },
		Write: func(c Ctx, _ bool) error {
			path := filepath.Join(c.Home, ".codex", "config.toml")
			b, _ := os.ReadFile(path)
			s := string(b)
			if i := strings.Index(s, codexBegin); i >= 0 {
				if j := strings.Index(s[i:], codexEnd); j >= 0 {
					s = s[:i] + strings.TrimLeft(s[i+j+len(codexEnd):], "\n")
				}
			}
			if s != "" && !strings.HasSuffix(s, "\n") {
				s += "\n"
			}
			if s != "" {
				s += "\n"
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			return os.WriteFile(path, []byte(s+codexBlock(c)), 0o600)
		},
	})
}
