package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/shashankxrm/deskdrop/internal/agent"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		args = []string{"run"}
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "pair":
		return cmdPair(rest)
	case "invite":
		return cmdInvite(rest)
	case "run":
		return cmdRun(rest)
	case "install":
		return cmdInstall(rest)
	case "uninstall":
		return cmdUninstall(rest)
	case "-h", "--help", "help":
		fmt.Fprint(os.Stderr, usage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n%s", cmd, usage)
		return 2
	}
}

const usage = `deskdrop-agent: pair a Mac as a laptop device, poll pending links, open the default browser, ack.

  deskdrop-agent pair [-api URL] [-code CODE] [-name NAME] [-config PATH]
      default -api is https://shashank-dropdesk.duckdns.org (or DESKDROP_API)
  deskdrop-agent invite [-config PATH]
  deskdrop-agent run  [-config PATH] [-once] [-poll 10s] [-open-file PATH]
  deskdrop-agent install [-config PATH]
  deskdrop-agent uninstall

Config is 0600 under the OS user config dir (macOS: ~/Library/Application Support/deskdrop/config.json).
Unsigned binary: System Settings → Privacy & Security → Open Anyway.
`

func defaultAPI() string {
	if v := strings.TrimSpace(os.Getenv("DESKDROP_API")); v != "" {
		return v
	}
	return "https://shashank-dropdesk.duckdns.org"
}

func configFlag(fs *flag.FlagSet) *string {
	return fs.String("config", "", "path to config.json (default: user config dir)")
}

func resolveConfig(p string) (string, error) {
	if strings.TrimSpace(p) != "" {
		return p, nil
	}
	return agent.DefaultPath()
}

func cmdPair(args []string) int {
	fs := flag.NewFlagSet("pair", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	api := fs.String("api", defaultAPI(), "API base URL")
	code := fs.String("code", "", "pairing code or deskdrop:https://host|CODE QR payload")
	name := fs.String("name", "", "device name (default: hostname)")
	cfgPath := configFlag(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	path, err := resolveConfig(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	c := strings.TrimSpace(*code)
	if c == "" {
		fmt.Fprint(os.Stderr, "pairing code: ")
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && err != io.EOF {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		c = line
	}
	inv := agent.ParseInvite(c)
	apiBase := strings.TrimRight(*api, "/")
	if inv.API != "" {
		apiBase = inv.API
	}
	if !agent.CodeComplete(inv.Code) {
		fmt.Fprintln(os.Stderr, "need an 8-character pairing code")
		return 1
	}
	devName := strings.TrimSpace(*name)
	if devName == "" {
		devName, _ = os.Hostname()
		if devName == "" {
			devName = "Mac"
		}
	}
	cli := &agent.Client{Base: apiBase}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	got, err := cli.Claim(ctx, inv.Code, devName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cfg := agent.Config{
		APIBase:    cli.Base,
		Token:      got.Token,
		DeviceID:   got.DeviceID,
		DeviceName: devName,
	}
	if err := agent.Save(path, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("paired device_id=%s config=%s\n", got.DeviceID, path)
	return 0
}

func cmdInvite(args []string) int {
	fs := flag.NewFlagSet("invite", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	cfgPath := configFlag(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	path, err := resolveConfig(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cfg, err := agent.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "not paired (%v); run: deskdrop-agent pair\n", err)
		return 1
	}
	cli := &agent.Client{Base: cfg.APIBase, Token: cfg.Token}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	code, err := cli.CreatePairing(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(code)
	return 0
}

func cmdRun(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	cfgPath := configFlag(fs)
	once := fs.Bool("once", false, "one poll cycle then exit (E2E)")
	poll := fs.Duration("poll", 10*time.Second, "poll interval (sleep/wake guarantee)")
	openFile := fs.String("open-file", "", "append opened URLs here instead of the browser")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	path, err := resolveConfig(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cfg, err := agent.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "not paired (%v); run: deskdrop-agent pair\n", err)
		return 1
	}
	open := defaultOpener()
	if *openFile != "" {
		open = fileOpener(*openFile)
	}
	cli := &agent.Client{Base: cfg.APIBase, Token: cfg.Token}
	r := agent.NewRunner(cli, open)
	r.OnOpened = func(l agent.Link) {
		cfg.Remember(l.ID, l.URL, l.Title, time.Now())
		_ = agent.Save(path, cfg)
		fmt.Printf("opened %s\n", l.URL)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *once {
		if err := r.Cycle(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}

	if runtime.GOOS == "darwin" && *openFile == "" {
		return runTray(ctx, r, *poll, cfg, path)
	}
	if err := r.Run(ctx, *poll); err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func fileOpener(path string) agent.Opener {
	return func(_ context.Context, rawURL string) error {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = fmt.Fprintln(f, rawURL)
		return err
	}
}

func defaultOpener() agent.Opener {
	return func(ctx context.Context, rawURL string) error {
		var cmd *exec.Cmd
		if runtime.GOOS == "darwin" {
			cmd = exec.CommandContext(ctx, "open", rawURL)
		} else {
			cmd = exec.CommandContext(ctx, "xdg-open", rawURL)
		}
		return cmd.Run()
	}
}

func cmdInstall(args []string) int {
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "LaunchAgent install is macOS only")
		return 1
	}
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	cfgPath := configFlag(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	path, err := resolveConfig(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	plist, err := agent.LaunchAgentPlist(exe, path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	dest := launchAgentPath()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := os.WriteFile(dest, plist, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_ = exec.Command("launchctl", "unload", dest).Run()
	out, err := exec.Command("launchctl", "load", dest).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "launchctl load: %v\n%s\n", err, out)
		return 1
	}
	fmt.Printf("installed %s (RunAtLoad)\n", dest)
	return 0
}

func cmdUninstall(args []string) int {
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "LaunchAgent uninstall is macOS only")
		return 1
	}
	dest := launchAgentPath()
	_ = exec.Command("launchctl", "unload", dest).Run()
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("removed %s\n", dest)
	return 0
}

func launchAgentPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", agent.LaunchAgentLabel+".plist")
}
