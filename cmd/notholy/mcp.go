package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// mcpBridge is `notholy mcp`: newline-delimited JSON-RPC on stdio, forwarded to the remote MCP endpoint with the
// account key. The tools are defined once, on the server. The only local step: analyze_file with a path reads
// that file here and sends its bytes, because the server cannot see the user's disk.
func mcpBridge(ctx context.Context, in io.Reader, out io.Writer) error {
	c, _, err := client()
	if err != nil {
		return err
	}
	origin := c.Base
	if len(origin) > 3 && origin[len(origin)-3:] == "/v1" {
		origin = origin[:len(origin)-3]
	}
	var mu sync.Mutex
	write := func(b []byte) {
		mu.Lock()
		defer mu.Unlock()
		_, _ = out.Write(append(b, '\n'))
	}
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 64<<10), 64<<20)
	var wg sync.WaitGroup
	for sc.Scan() {
		line := append([]byte(nil), sc.Bytes()...)
		if len(line) == 0 {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, id, err := localStep(line)
			if err != nil {
				write(rpcError(id, -32602, err.Error()))
				return
			}
			c2 := *c
			c2.Base = origin
			status, body, err := c2.Raw(ctx, "/mcp", msg)
			switch {
			case err != nil:
				write(rpcError(id, -32603, "NotHoly is unreachable: "+err.Error()))
			case status == 202 || len(body) == 0:
				// notification accepted, nothing to send back
			case status >= 400 && id != nil:
				write(rpcError(id, -32603, fmt.Sprintf("NotHoly returned %d", status)))
			default:
				write(body)
			}
		}()
	}
	wg.Wait()
	return sc.Err()
}

// localStep turns analyze_file {path} into {name, content_base64}. It returns the request id for errors.
func localStep(line []byte) ([]byte, json.RawMessage, error) {
	var m struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		} `json:"params"`
	}
	if json.Unmarshal(line, &m) != nil || m.Method != "tools/call" || m.Params.Name != "analyze_file" {
		return line, m.ID, nil
	}
	p, _ := m.Params.Arguments["path"].(string)
	if p == "" {
		return line, m.ID, nil
	}
	st, err := os.Stat(p)
	if err != nil {
		return nil, m.ID, err
	}
	if st.Size() > 25<<20 {
		return nil, m.ID, fmt.Errorf("%s is larger than 25 MB", p)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, m.ID, err
	}
	var full map[string]any
	_ = json.Unmarshal(line, &full)
	args := map[string]any{"name": filepath.Base(p), "content_base64": base64.StdEncoding.EncodeToString(b)}
	full["params"].(map[string]any)["arguments"] = args
	out, err := json.Marshal(full)
	return out, m.ID, err
}

func rpcError(id json.RawMessage, code int, msg string) []byte {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": msg}})
	return b
}
