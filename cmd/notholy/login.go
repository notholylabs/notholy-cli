package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/notholylabs/notholy-cli/internal/api"
	"github.com/notholylabs/notholy-cli/internal/config"
)

// login runs the device flow (RFC 8628): show a code, open /activate, poll until the user approves there.
func login(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	noBrowser := fs.Bool("no-browser", false, "print the link instead of opening a browser")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg := config.Load()
	host, _ := os.Hostname()
	var code struct {
		DeviceCode string `json:"device_code"`
		UserCode   string `json:"user_code"`
		URI        string `json:"verification_uri"`
		Complete   string `json:"verification_uri_complete"`
		ExpiresIn  int    `json:"expires_in"`
		Interval   int    `json:"interval"`
	}
	if err := post(ctx, cfg.AppURL+"/api/cli/device/code", map[string]string{"client_name": "notholy CLI", "host": host}, &code, nil); err != nil {
		return fmt.Errorf("could not start sign-in: %w", err)
	}
	_, _ = fmt.Fprintf(stdout, "%s Confirm this code in your browser: %s\n  %s\n", color("✦", red), code.UserCode, code.Complete)
	if !*noBrowser && os.Getenv("NOTHOLY_NO_BROWSER") == "" {
		openBrowser(code.Complete)
	}
	interval := time.Duration(max(code.Interval, 1)) * time.Second
	deadline := time.Now().Add(time.Duration(code.ExpiresIn) * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
		var ok struct {
			Key     string `json:"api_key"`
			APIBase string `json:"api_base"`
			Org     struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"organization"`
		}
		var fail struct {
			Error string `json:"error"`
		}
		err := post(ctx, cfg.AppURL+"/api/cli/device/token", map[string]string{"device_code": code.DeviceCode}, &ok, &fail)
		if err == nil && ok.Key != "" {
			fallback, err := config.StoreKey(ok.Key)
			if err != nil {
				return err
			}
			cfg.OrgID, cfg.OrgName = ok.Org.ID, ok.Org.Name
			if ok.APIBase != "" && os.Getenv("NOTHOLY_API_BASE") == "" {
				cfg.APIBase = ok.APIBase
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			where := "the system keychain"
			if fallback {
				where = config.Dir() + string(os.PathSeparator) + "credentials (mode 600; no keychain was available)"
			}
			_, _ = fmt.Fprintf(stdout, "%s Signed in to %s. Key %s saved in %s.\n", color("✦", red), ok.Org.Name, config.Mask(ok.Key), where)
			return nil
		}
		switch fail.Error {
		case "authorization_pending":
		case "slow_down":
			interval += 5 * time.Second
		case "access_denied":
			return errors.New("sign-in was denied in the browser")
		case "expired_token":
			return errors.New("the code expired; run notholy login again")
		default:
			if err != nil {
				return err
			}
		}
	}
	return errors.New("the code expired; run notholy login again")
}

// post sends JSON; a 4xx with a JSON body is decoded into fail and is not an error.
func post(ctx context.Context, url string, body, out, fail any) error {
	b, _ := json.Marshal(body)
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	r.Header.Set("content-type", "application/json")
	r.Header.Set("user-agent", api.UserAgent())
	res, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode >= 400 && res.StatusCode < 500 && fail != nil {
		return json.NewDecoder(res.Body).Decode(fail)
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("http %d", res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func openBrowser(u string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	case "darwin":
		cmd = exec.Command("open", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	_ = cmd.Start()
}
