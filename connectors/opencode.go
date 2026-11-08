package connectors

import (
	"encoding/json"
	"path/filepath"
)

// OpenCode uses an OpenAI-compatible provider. `run` passes the whole config in OPENCODE_CONFIG_CONTENT;
// `connect` merges the provider into ~/.config/opencode/opencode.json. The key is read from NOTHOLY_API_KEY.
func opencodeProvider(c Ctx) map[string]any {
	model := func(name string) map[string]any {
		return map[string]any{"name": name, "tool_call": true, "reasoning": true, "limit": map[string]any{"context": 1048576, "output": 131072}}
	}
	return map[string]any{
		"npm":     "@ai-sdk/openai-compatible",
		"name":    "NotHoly",
		"options": map[string]any{"baseURL": c.APIBase, "apiKey": "{env:NOTHOLY_API_KEY}"},
		"models": map[string]any{
			"notholy-specter": model("Specter"),
			"notholy-pulse":   model("Pulse Flash"),
			"notholy-vector":  model("Vector Flash"),
		},
	}
}

func init() {
	register(&Connector{
		Name: "opencode",
		Bin:  "opencode",
		Env: func(c Ctx) map[string]string {
			cfg, _ := json.Marshal(map[string]any{
				"$schema":  "https://opencode.ai/config.json",
				"model":    "notholy/" + c.Model,
				"provider": map[string]any{"notholy": opencodeProvider(c)},
			})
			return map[string]string{"NOTHOLY_API_KEY": c.Key, "OPENCODE_CONFIG_CONTENT": string(cfg)}
		},
		Files: func(c Ctx, _ bool) []string {
			return []string{filepath.Join(c.Home, ".config", "opencode", "opencode.json")}
		},
		Write: func(c Ctx, _ bool) error {
			path := filepath.Join(c.Home, ".config", "opencode", "opencode.json")
			m, err := readJSON(path)
			if err != nil {
				return err
			}
			if _, ok := m["$schema"]; !ok {
				m["$schema"] = "https://opencode.ai/config.json"
			}
			obj(m, "provider")["notholy"] = opencodeProvider(c)
			if _, ok := m["model"]; !ok {
				m["model"] = "notholy/" + c.Model
			}
			return writeJSON(path, m)
		},
	})
}
