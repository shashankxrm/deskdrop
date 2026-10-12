package agent

import (
	"encoding/xml"
	"fmt"
	"strings"
)

const LaunchAgentLabel = "com.deskdrop.agent"

func LaunchAgentPlist(execPath, configPath string) ([]byte, error) {
	if execPath == "" {
		return nil, fmt.Errorf("executable path required")
	}
	type item struct {
		XMLName xml.Name
		Text    string `xml:",chardata"`
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + xmlEscape(LaunchAgentLabel) + `</string>
	<key>ProgramArguments</key>
	<array>
		<string>` + xmlEscape(execPath) + `</string>
		<string>run</string>
		<string>-config</string>
		<string>` + xmlEscape(configPath) + `</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<dict>
		<key>SuccessfulExit</key>
		<false/>
	</dict>
	<key>ProcessType</key>
	<string>Interactive</string>
</dict>
</plist>
`)
	return []byte(b.String()), nil
}

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
