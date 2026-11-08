// Package api talks to the NotHoly gateway: JSON calls, file upload and streamed chat completions.
package api

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

var Version = "dev"

type Client struct {
	Base string
	Key  string
	HTTP *http.Client
}

func New(base, key string) *Client {
	return &Client{Base: strings.TrimRight(base, "/"), Key: key, HTTP: &http.Client{Timeout: 10 * time.Minute}}
}

// DefaultModel is the catalog's default chat model: the one /models marks default, else the first listed. No model
// name is fixed in the CLI; the catalog decides.
func (c *Client) DefaultModel(ctx context.Context) (string, error) {
	var r struct {
		Data []struct {
			ID      string `json:"id"`
			Default bool   `json:"default"`
		} `json:"data"`
	}
	if err := c.JSON(ctx, http.MethodGet, "/models", nil, &r); err != nil {
		return "", err
	}
	for _, m := range r.Data {
		if m.Default {
			return m.ID, nil
		}
	}
	if len(r.Data) > 0 {
		return r.Data[0].ID, nil
	}
	return "", errors.New("the model catalog is empty")
}

func UserAgent() string { return fmt.Sprintf("notholy-cli/%s (%s; %s)", Version, runtime.GOOS, runtime.GOARCH) }

// Error is a gateway error with its public code and message.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s (%d): %s", e.Code, e.Status, e.Message)
	}
	return fmt.Sprintf("http %d", e.Status)
}

func readError(res *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	var e struct {
		Error struct {
			Code    string `json:"code"`
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(b, &e)
	code := e.Error.Code
	if code == "" {
		code = e.Error.Type
	}
	return &Error{Status: res.StatusCode, Code: code, Message: e.Error.Message}
}

func (c *Client) req(ctx context.Context, method, path string, body io.Reader, ctype string) (*http.Response, error) {
	r, err := http.NewRequestWithContext(ctx, method, c.Base+path, body)
	if err != nil {
		return nil, err
	}
	if c.Key != "" {
		r.Header.Set("authorization", "Bearer "+c.Key)
	}
	r.Header.Set("user-agent", UserAgent())
	if ctype != "" {
		r.Header.Set("content-type", ctype)
	}
	return c.HTTP.Do(r)
}

// JSON sends v (nil for GET) and decodes the reply into out.
func (c *Client) JSON(ctx context.Context, method, path string, v, out any) error {
	var body io.Reader
	ctype := ""
	if v != nil {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		body, ctype = bytes.NewReader(b), "application/json"
	}
	res, err := c.req(ctx, method, path, body, ctype)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode >= 300 {
		return readError(res)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// Raw posts a body and returns the raw reply, for the MCP bridge.
func (c *Client) Raw(ctx context.Context, path string, body []byte) (int, []byte, error) {
	res, err := c.req(ctx, http.MethodPost, path, bytes.NewReader(body), "application/json")
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = res.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	return res.StatusCode, b, err
}

func (c *Client) Upload(ctx context.Context, name string, data []byte, out any) error {
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Base+"/files?wait=1", bytes.NewReader(data))
	if err != nil {
		return err
	}
	r.Header.Set("authorization", "Bearer "+c.Key)
	r.Header.Set("user-agent", UserAgent())
	r.Header.Set("content-type", "application/octet-stream")
	r.Header.Set("x-file-name", name)
	res, err := c.HTTP.Do(r)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode >= 300 {
		return readError(res)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// ---------- chat completions ----------

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type Tool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters"`
	} `json:"function"`
}

type ChatRequest struct {
	Model           string    `json:"model"`
	Messages        []Message `json:"messages"`
	Tools           []Tool    `json:"tools,omitempty"`
	ReasoningEffort string    `json:"reasoning_effort,omitempty"`
	Stream          bool      `json:"stream"`
	StreamOptions   any       `json:"stream_options,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type Result struct {
	Message Message
	Finish  string
	Usage   Usage
}

// Stream runs one streamed completion, calling onText for each text piece and onReason for reasoning.
// Tool calls arrive in pieces and are put back together by index.
func (c *Client) Stream(ctx context.Context, req ChatRequest, onText, onReason func(string)) (*Result, error) {
	req.Stream = true
	req.StreamOptions = map[string]bool{"include_usage": true}
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	res, err := c.req(ctx, http.MethodPost, "/chat/completions", bytes.NewReader(b), "application/json")
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode >= 300 {
		return nil, readError(res)
	}
	out := &Result{Message: Message{Role: "assistant"}}
	calls := map[int]*ToolCall{}
	order := []int{}
	sc := bufio.NewScanner(res.Body)
	sc.Buffer(make([]byte, 64<<10), 8<<20)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var ch struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
					Reasoning        string `json:"reasoning"`
					ToolCalls        []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *Usage `json:"usage"`
			Error *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(data), &ch) != nil {
			continue
		}
		if ch.Error != nil {
			return out, &Error{Status: 502, Code: ch.Error.Code, Message: ch.Error.Message}
		}
		if ch.Usage != nil {
			out.Usage = *ch.Usage
		}
		for _, choice := range ch.Choices {
			d := choice.Delta
			if r := d.ReasoningContent + d.Reasoning; r != "" && onReason != nil {
				onReason(r)
			}
			if d.Content != "" {
				out.Message.Content += d.Content
				if onText != nil {
					onText(d.Content)
				}
			}
			for _, tc := range d.ToolCalls {
				t, ok := calls[tc.Index]
				if !ok {
					t = &ToolCall{Type: "function"}
					calls[tc.Index] = t
					order = append(order, tc.Index)
				}
				if tc.ID != "" {
					t.ID = tc.ID
				}
				t.Function.Name += tc.Function.Name
				t.Function.Arguments += tc.Function.Arguments
			}
			if choice.FinishReason != nil {
				out.Finish = *choice.FinishReason
			}
		}
	}
	if err := sc.Err(); err != nil {
		return out, err
	}
	for _, i := range order {
		out.Message.ToolCalls = append(out.Message.ToolCalls, *calls[i])
	}
	return out, nil
}
