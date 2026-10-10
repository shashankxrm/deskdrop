package config

import (
	"fmt"
	"os"
	"strings"
)

// Config is process configuration. Secrets have no defaults; Load fails if they are empty.
type Config struct {
	HTTPAddr           string
	DatabaseURL        string
	DeviceTokenPepper  string
}

func Load() (Config, error) {
	c := Config{
		HTTPAddr:          strings.TrimSpace(os.Getenv("HTTP_ADDR")),
		DatabaseURL:       strings.TrimSpace(os.Getenv("DATABASE_URL")),
		DeviceTokenPepper: strings.TrimSpace(os.Getenv("DEVICE_TOKEN_PEPPER")),
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
