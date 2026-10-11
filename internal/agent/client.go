package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	Base  string
	Token string
	HTTP  *http.Client
	Sleep func(context.Context, time.Duration) error
}

type Link struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Title     string `json:"title,omitempty"`
	CreatedAt string `json:"created_at"`
}

type ClaimResult struct {
	Token    string `json:"token"`
	DeviceID string `json:"device_id"`
	Type     string `json:"type"`
}

type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("http %d", e.Status)
	}
	return fmt.Sprintf("http %d: %s", e.Status, e.Message)
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (c *Client) CreatePairing(ctx context.Context) (string, error) {
	var out struct {
		Code string `json:"code"`
	}
	err := c.doJSON(ctx, http.MethodPost, "/v1/pairings", []byte("{}"), c.Token != "", &out)
	return out.Code, err
}

func (c *Client) Claim(ctx context.Context, code, name string) (ClaimResult, error) {
	code = NormalizeCode(code)
	body, err := json.Marshal(map[string]string{
		"code": code,
		"type": "laptop",
		"name": name,
	})
	if err != nil {
		return ClaimResult{}, err
	}
	var out ClaimResult
	err = c.doJSON(ctx, http.MethodPost, "/v1/pairings/claim", body, false, &out)
	return out, err
}

func (c *Client) Pending(ctx context.Context) ([]Link, error) {
	var wrap struct {
		Links []Link `json:"links"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/v1/links/pending", nil, true, &wrap)
	if wrap.Links == nil {
		wrap.Links = []Link{}
	}
	return wrap.Links, err
}

func (c *Client) Ack(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodPost, "/v1/links/"+id+"/ack", nil, true, nil)
}

func (c *Client) doJSON(ctx context.Context, method, path string, body []byte, auth bool, dest any) error {
	var last error
	for attempt := 1; attempt <= 5; attempt++ {
		err := c.doOnce(ctx, method, path, body, auth, dest)
		if err == nil {
			return nil
		}
		var api *APIError
		if !errors.As(err, &api) {
			return err
		}
		if !ShouldRetry(api.Status) {
			return err
		}
		// Do not replay POSTs on a lost response (claim is single-use).
		if method != http.MethodGet && api.Status == 0 {
			return err
		}
		last = err
		if attempt == 5 {
			break
		}
		wait := FullJitter(attempt, 200*time.Millisecond, 5*time.Second)
		sleep := c.Sleep
		if sleep == nil {
			sleep = func(ctx context.Context, d time.Duration) error {
				t := time.NewTimer(d)
				defer t.Stop()
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-t.C:
					return nil
				}
			}
		}
		if err := sleep(ctx, wait); err != nil {
			return err
		}
	}
	return last
}

func (c *Client) doOnce(ctx context.Context, method, path string, body []byte, auth bool, dest any) error {
	u := strings.TrimRight(c.Base, "/") + path
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	res, err := c.http().Do(req)
	if err != nil {
		return &APIError{Status: 0, Message: err.Error()}
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 400 {
		msg := strings.TrimSpace(string(b))
		var wrap struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(b, &wrap) == nil && wrap.Error != "" {
			msg = wrap.Error
		}
		return &APIError{Status: res.StatusCode, Message: msg}
	}
	if dest == nil || res.StatusCode == http.StatusNoContent || len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, dest)
}
