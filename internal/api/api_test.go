package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStreamAssemblesTextReasoningToolCallsAndUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("authorization") != "Bearer k" {
			w.WriteHeader(401)
			_, _ = fmt.Fprint(w, `{"error":{"code":"invalid_api_key","message":"bad key"}}`)
			return
		}
		w.Header().Set("content-type", "text/event-stream")
		for _, l := range []string{
			`{"choices":[{"delta":{"reasoning_content":"hmm"}}]}`,
			`{"choices":[{"delta":{"content":"Hel"}}]}`,
			`{"choices":[{"delta":{"content":"lo"}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","type":"function","function":{"name":"bash","arguments":"{\"comm"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"and\":\"ls\"}"}}]}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
			`{"choices":[],"usage":{"prompt_tokens":12,"completion_tokens":5}}`,
		} {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", l)
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	text, reason := "", ""
	res, err := New(srv.URL, "k").Stream(context.Background(), ChatRequest{Model: "m"}, func(s string) { text += s }, func(s string) { reason += s })
	if err != nil {
		t.Fatal(err)
	}
	if text != "Hello" || reason != "hmm" || res.Finish != "tool_calls" || res.Usage.PromptTokens != 12 {
		t.Fatalf("got %q %q %+v", text, reason, res)
	}
	if len(res.Message.ToolCalls) != 1 || res.Message.ToolCalls[0].Function.Arguments != `{"command":"ls"}` || res.Message.ToolCalls[0].ID != "c1" {
		t.Fatalf("tool call: %+v", res.Message.ToolCalls)
	}

	_, err = New(srv.URL, "wrong").Stream(context.Background(), ChatRequest{Model: "m"}, nil, nil)
	e, ok := err.(*Error)
	if !ok || e.Status != 401 || e.Code != "invalid_api_key" {
		t.Fatalf("error: %v", err)
	}
}
