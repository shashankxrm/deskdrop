package config

import "testing"

func TestLoad_required(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
		addr    string
	}{
		{
			name:    "missing pepper",
			env:     map[string]string{"DATABASE_URL": "postgres://u:p@localhost:5432/db"},
			wantErr: true,
		},
		{
			name:    "missing database url",
			env:     map[string]string{"DEVICE_TOKEN_PEPPER": "pepper-value-for-tests"},
			wantErr: true,
		},
		{
			name: "missing both",
			env:  map[string]string{},
			wantErr: true,
		},
		{
			name: "ok with default addr",
			env: map[string]string{
				"DEVICE_TOKEN_PEPPER": "pepper-value-for-tests",
				"DATABASE_URL":        "postgres://u:p@localhost:5432/db",
			},
			wantErr: false,
			addr:    ":8080",
		},
		{
			name: "ok with explicit addr",
			env: map[string]string{
				"DEVICE_TOKEN_PEPPER": "pepper-value-for-tests",
				"DATABASE_URL":        "postgres://u:p@localhost:5432/db",
				"HTTP_ADDR":           ":9999",
			},
			wantErr: false,
			addr:    ":9999",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DEVICE_TOKEN_PEPPER", "")
			t.Setenv("DATABASE_URL", "")
			t.Setenv("HTTP_ADDR", "")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			cfg, err := Load()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.HTTPAddr != tc.addr {
				t.Fatalf("HTTPAddr=%q want %q", cfg.HTTPAddr, tc.addr)
			}
			if cfg.DeviceTokenPepper != "pepper-value-for-tests" {
				t.Fatalf("pepper not loaded")
			}
		})
	}
}
