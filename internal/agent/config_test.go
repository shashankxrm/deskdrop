package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveLoad_mode0600(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deskdrop", "config.json")
	c := Config{APIBase: "http://127.0.0.1:8080/", Token: "abc", DeviceID: "d1", DeviceName: "mac"}
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("file perm %o want 0600", st.Mode().Perm())
	}
	dst, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if dst.Mode().Perm() != 0o700 {
		t.Fatalf("dir perm %o want 0700", dst.Mode().Perm())
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.APIBase != "http://127.0.0.1:8080" {
		t.Fatalf("api_base %q", got.APIBase)
	}
	if got.Token != "abc" || got.DeviceID != "d1" {
		t.Fatalf("got %+v", got)
	}
}

func TestRemember_capsAndDedupe(t *testing.T) {
	var c Config
	for i := 0; i < 12; i++ {
		c.Remember(string(rune('a'+i)), "https://example.com/"+string(rune('a'+i)), "", time.Unix(int64(i), 0))
	}
	if len(c.Recent) != maxRecent {
		t.Fatalf("len %d", len(c.Recent))
	}
	if c.Recent[0].ID != string(rune('a'+11)) {
		t.Fatalf("newest %s", c.Recent[0].ID)
	}
	c.Remember("l", "https://example.com/l", "t", time.Unix(99, 0))
	c.Remember("l", "https://example.com/l", "t2", time.Unix(100, 0))
	n := 0
	for _, r := range c.Recent {
		if r.ID == "l" {
			n++
			if r.Title != "t2" {
				t.Fatalf("title %s", r.Title)
			}
		}
	}
	if n != 1 {
		t.Fatalf("dupes %d", n)
	}
}

func TestNormalizeCode(t *testing.T) {
	if got := NormalizeCode(" abcd efgh "); got != "ABCD-EFGH" {
		t.Fatalf("got %q", got)
	}
	if !CodeComplete("abcd-efgh") || CodeComplete("abc") {
		t.Fatal("complete")
	}
	inv := ParseInvite("https://shashank-dropdesk.duckdns.org/connect?code=abcd-efgh")
	if inv.API != "https://shashank-dropdesk.duckdns.org" || inv.Code != "ABCD-EFGH" {
		t.Fatalf("%+v", inv)
	}
	pipe := ParseInvite("deskdrop:https://shashank-dropdesk.duckdns.org|abcd-efgh")
	if pipe.API != "https://shashank-dropdesk.duckdns.org" || pipe.Code != "ABCD-EFGH" {
		t.Fatalf("%+v", pipe)
	}
}

func TestLaunchAgentPlist(t *testing.T) {
	b, err := LaunchAgentPlist("/opt/deskdrop/deskdrop-agent", "/tmp/cfg.json")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{LaunchAgentLabel, "/opt/deskdrop/deskdrop-agent", "-config", "/tmp/cfg.json", "RunAtLoad"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in %s", want, s)
		}
	}
	b2, err := LaunchAgentPlist("/tmp/a&b", "/tmp/c<d>")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b2), "<d>") || !strings.Contains(string(b2), "a&amp;b") {
		t.Fatalf("unescaped: %s", b2)
	}
}
