//go:build darwin

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"fyne.io/systray"
	"github.com/shashankxrm/deskdrop/internal/agent"
)

func runTray(ctx context.Context, r *agent.Runner, poll time.Duration, cfg agent.Config, cfgPath string) int {
	code := 0
	systray.Run(func() {
		systray.SetTitle("DD")
		systray.SetTooltip("deskdrop")
		status := systray.AddMenuItem("polling…", "")
		status.Disable()
		systray.AddSeparator()
		recentItems := make([]*systray.MenuItem, 8)
		for i := range recentItems {
			recentItems[i] = systray.AddMenuItem("-", "")
			recentItems[i].Hide()
		}
		refreshRecent := func(list []agent.Recent) {
			for i, item := range recentItems {
				if i >= len(list) {
					item.Hide()
					continue
				}
				label := list[i].URL
				if list[i].Title != "" {
					label = list[i].Title
				}
				item.SetTitle(label)
				item.Show()
			}
		}
		refreshRecent(cfg.Recent)
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Stop deskdrop")

		var mu sync.Mutex
		go func() {
			for {
				select {
				case <-ctx.Done():
					systray.Quit()
					return
				case <-mQuit.ClickedCh:
					systray.Quit()
					return
				}
			}
		}()
		for i, item := range recentItems {
			i, item := i, item
			go func() {
				for range item.ClickedCh {
					mu.Lock()
					list := cfg.Recent
					mu.Unlock()
					if i < len(list) {
						_ = exec.Command("open", list[i].URL).Start()
					}
				}
			}()
		}

		orig := r.OnOpened
		r.OnOpened = func(l agent.Link) {
			if orig != nil {
				orig(l)
			}
			mu.Lock()
			_ = cfg // cfg updated in orig via Remember+Save
			loaded, err := agent.Load(cfgPath)
			if err == nil {
				cfg = loaded
				refreshRecent(cfg.Recent)
			}
			mu.Unlock()
			status.SetTitle("last: " + l.URL)
		}

		go func() {
			if err := r.Run(ctx, poll); err != nil && err != context.Canceled {
				fmt.Fprintln(os.Stderr, err)
				status.SetTitle("error: " + err.Error())
				code = 1
				systray.Quit()
			}
		}()
	}, func() {})
	return code
}
