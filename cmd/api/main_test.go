package main

import "testing"

func TestRun_missingEnv(t *testing.T) {
	t.Setenv("DEVICE_TOKEN_PEPPER", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("HTTP_ADDR", "")
	if code := run(nil); code != 1 {
		t.Fatalf("run()=%d want 1", code)
	}
}

func TestRun_missingPepperOnly(t *testing.T) {
	t.Setenv("DEVICE_TOKEN_PEPPER", "")
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	if code := run(nil); code != 1 {
		t.Fatalf("run()=%d want 1", code)
	}
}
