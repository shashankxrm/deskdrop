package agent

import (
	"net/url"
	"strings"
)

func NormalizeCode(raw string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(raw)) {
		if r == '-' || r == ' ' {
			continue
		}
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if len(s) == 8 {
		return s[:4] + "-" + s[4:]
	}
	return s
}

func CodeComplete(raw string) bool {
	return len(ParseInvite(raw).Code) == 9
}

// Invite is a pairing code, optionally with an API base (from a dashboard QR).
type Invite struct {
	API  string
	Code string
}

func ParseInvite(raw string) Invite {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, "%7C", "|")
	s = strings.ReplaceAll(s, "%7c", "|")
	s = strings.TrimPrefix(s, "deskdrop:")
	s = strings.TrimPrefix(s, "DESKDROP:")
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		u, err := url.Parse(s)
		if err == nil && u.Host != "" {
			api := u.Scheme + "://" + u.Host
			code := u.Query().Get("code")
			if code == "" {
				code = u.Fragment
			}
			if code == "" {
				if i := strings.LastIndex(s, "|"); i > 0 {
					code = s[i+1:]
				}
			}
			return Invite{API: api, Code: NormalizeCode(code)}
		}
	}
	if i := strings.LastIndex(s, "|"); i > 0 {
		left := strings.TrimRight(strings.TrimSpace(s[:i]), "/")
		right := NormalizeCode(s[i+1:])
		if strings.HasPrefix(left, "http://") || strings.HasPrefix(left, "https://") {
			return Invite{API: left, Code: right}
		}
	}
	if strings.Contains(s, "://") {
		if i := strings.LastIndex(s, "#"); i > 0 {
			return Invite{API: strings.TrimRight(strings.TrimSpace(s[:i]), "/"), Code: NormalizeCode(s[i+1:])}
		}
		return Invite{}
	}
	return Invite{Code: NormalizeCode(s)}
}
