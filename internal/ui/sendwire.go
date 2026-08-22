package ui

import "strings"

// NormalizeTerminalSend converts user/YAML send text to what an interactive
// SSH/telnet session expects for Enter.
//
// The terminal widget sends KeyReturn as CR ("\r"). Button YAML and scripts
// conventionally write "\n". Sending LF alone to Cisco IOS and similar CLIs
// often yields a blank line, odd cursor placement, or a white paint streak
// instead of executing the command.
func NormalizeTerminalSend(s string) string {
	if s == "" {
		return s
	}
	// CRLF / bare CR → LF, then every LF → CR (same as typing Enter).
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\n", "\r")
	return s
}
