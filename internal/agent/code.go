package agent

import "strings"

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
	return len(NormalizeCode(raw)) == 9
}
