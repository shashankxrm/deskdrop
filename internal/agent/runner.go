package agent

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Opener opens a URL in the default browser (or a test fake).
type Opener func(ctx context.Context, rawURL string) error

type Runner struct {
	Client   *Client
	Open     Opener
	OnOpened func(Link)
	mu       sync.Mutex
	opened   map[string]bool // browser already opened this session; ack may still be in flight
}

func NewRunner(c *Client, open Opener) *Runner {
	return &Runner{Client: c, Open: open, opened: map[string]bool{}}
}

func (r *Runner) Cycle(ctx context.Context) error {
	links, err := r.Client.Pending(ctx)
	if err != nil {
		return err
	}
	for _, l := range links {
		if err := r.deliver(ctx, l); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	if err := r.Cycle(ctx); err != nil {
		return err
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if err := r.Cycle(ctx); err != nil {
				return err
			}
		}
	}
}

func (r *Runner) deliver(ctx context.Context, l Link) error {
	if err := assertHTTPURL(l.URL); err != nil {
		return err
	}
	r.mu.Lock()
	already := r.opened[l.ID]
	r.mu.Unlock()
	if !already {
		if err := r.Open(ctx, l.URL); err != nil {
			return fmt.Errorf("open: %w", err)
		}
		r.mu.Lock()
		r.opened[l.ID] = true
		r.mu.Unlock()
		if r.OnOpened != nil {
			r.OnOpened(l)
		}
	}
	return r.Client.Ack(ctx, l.ID)
}

func assertHTTPURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid url")
	}
	return nil
}
