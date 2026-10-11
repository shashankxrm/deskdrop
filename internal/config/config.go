package config

import (
	"fmt"
	"os"
	"strings"
)

// Config is process configuration. Secrets have no defaults; Load fails if they are empty.
type Config struct {
	HTTPAddr          string
	DatabaseURL       string
	DeviceTokenPepper string
	CORSOrigins       []string
	CookieSecure      bool
	TrustProxy        bool
}

func Load() (Config, error) {
	c := Config{
		HTTPAddr:          strings.TrimSpace(os.Getenv("HTTP_ADDR")),
		DatabaseURL:       strings.TrimSpace(os.Getenv("DATABASE_URL")),
		DeviceTokenPepper: strings.TrimSpace(os.Getenv("DEVICE_TOKEN_PEPPER")),
		CORSOrigins:       SplitCORSOrigins(os.Getenv("CORS_ORIGINS")),
		CookieSecure:      envBool("COOKIE_SECURE"),
		TrustProxy:        envBool("TRUST_PROXY"),
	}
	if c.HTTPAddr == "" {
		c.HTTPAddr = ":8080"
	}
	var missing []string
	if c.DeviceTokenPepper == "" {
		missing = append(missing, "DEVICE_TOKEN_PEPPER")
	}
	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}
	return c, nil
}

// SplitCORSOrigins parses a comma-separated Origin allowlist. Empty input is an empty list (no ACAO).
func SplitCORSOrigins(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envBool(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
}
