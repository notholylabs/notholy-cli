package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReorderPutsFlagsFirst(t *testing.T) {
	got := reorder([]string{"openssl", "--kev", "--min-cvss", "7", "heap"})
	want := []string{"--kev", "--min-cvss", "7", "openssl", "heap"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
}

func TestSplitJSON(t *testing.T) {
	a, js := split([]string{"cve", "--json", "CVE-1"})
	if !js || !reflect.DeepEqual(a, []string{"cve", "CVE-1"}) {
		t.Fatal(a, js)
	}
}

func TestMCPLocalStepReadsThePathHere(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.bin")
	_ = os.WriteFile(p, []byte("MZ\x00hello"), 0o600)
	line, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 7, "method": "tools/call", "params": map[string]any{"name": "analyze_file", "arguments": map[string]any{"path": p}}})
	out, id, err := localStep(line)
	if err != nil || string(id) != "7" {
		t.Fatal(err, string(id))
	}
	var m struct {
		Params struct {
			Arguments map[string]string `json:"arguments"`
		} `json:"params"`
	}
	_ = json.Unmarshal(out, &m)
	b, _ := base64.StdEncoding.DecodeString(m.Params.Arguments["content_base64"])
	if m.Params.Arguments["name"] != "a.bin" || string(b) != "MZ\x00hello" || m.Params.Arguments["path"] != "" {
		t.Fatalf("%s", out)
	}
	// other calls pass through untouched
	other := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	if got, _, _ := localStep(other); string(got) != string(other) {
		t.Fatal("changed a call it should not touch")
	}
}

func TestApprovalByMode(t *testing.T) {
	yes := func() *bufio.Reader { return bufio.NewReader(strings.NewReader("y\n")) }
	no := func() *bufio.Reader { return bufio.NewReader(strings.NewReader("\n")) }
	cases := []struct {
		mode, tool string
		in         *bufio.Reader
		want       bool
	}{
		{"plan", "write", yes(), false},
		{"ask", "bash", yes(), true},
		{"ask", "bash", no(), false},
		{"auto-edit", "edit", no(), true},
		{"auto-edit", "bash", no(), false},
		{"yolo", "bash", no(), true},
	}
	for _, c := range cases {
		if got := approve(c.mode, c.tool, "x", c.in); got != c.want {
			t.Errorf("%s/%s = %v", c.mode, c.tool, got)
		}
	}
}
