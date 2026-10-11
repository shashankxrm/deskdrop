//go:build !darwin

package main

import (
	"context"
	"time"

	"github.com/shashankxrm/deskdrop/internal/agent"
)

func runTray(ctx context.Context, r *agent.Runner, poll time.Duration, _ agent.Config, _ string) int {
	if err := r.Run(ctx, poll); err != nil && err != context.Canceled {
		return 1
	}
	return 0
}
