package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	configDirPerm  os.FileMode = 0o700
	configFilePerm os.FileMode = 0o600
	maxRecent                  = 10
)

// Config is the on-disk laptop identity. Token is plaintext on disk; the file must stay 0600.
type Config struct {
	APIBase    string   `json:"api_base"`
	Token      string   `json:"token"`
	DeviceID   string   `json:"device_id"`
	DeviceName string   `json:"device_name"`
	Recent     []Recent `json:"recent,omitempty"`
}

// Recent is a link the agent already opened (tray + relaunch history, not the queue).
type Recent struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Title    string `json:"title,omitempty"`
	OpenedAt string `json:"opened_at"`
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "deskdrop", "config.json"), nil
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	c.APIBase = strings.TrimRight(strings.TrimSpace(c.APIBase), "/")
	if c.APIBase == "" || c.Token == "" {
		return Config{}, fmt.Errorf("config: api_base and token required")
	}
	return c, nil
}

func Save(path string, c Config) error {
	c.APIBase = strings.TrimRight(strings.TrimSpace(c.APIBase), "/")
	if len(c.Recent) > maxRecent {
		c.Recent = c.Recent[:maxRecent]
	}
	if err := os.MkdirAll(filepath.Dir(path), configDirPerm); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Dir(path), configDirPerm); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, configFilePerm); err != nil {
		return err
	}
	if err := os.Chmod(tmp, configFilePerm); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Chmod(path, configFilePerm)
}

func (c *Config) Remember(id, url, title string, at time.Time) {
	item := Recent{ID: id, URL: url, Title: title, OpenedAt: at.UTC().Format(time.RFC3339)}
	out := []Recent{item}
	for _, r := range c.Recent {
		if r.ID == id {
			continue
		}
		out = append(out, r)
		if len(out) == maxRecent {
			break
		}
	}
	c.Recent = out
}
