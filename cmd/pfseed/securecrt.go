package main

import (
	"github.com/scottpeterman/pathfinderssh/internal/crtimport"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// Thin aliases so existing pfseed call sites keep compiling during the move.
type crtSession = crtimport.Session

func importSecureCRT(configRoot string) ([]crtSession, error) {
	return crtimport.Import(configRoot)
}

func crtToNode(cs crtSession) sessions.Node {
	return crtimport.ToNode(cs)
}

func parseCRTSessionINI(path, rel string) (crtSession, error) {
	return crtimport.ParseSessionINI(path, rel)
}

func splitCRTField(line string) (key, val string, ok bool) {
	eq := indexByte(line, '=')
	if eq < 0 {
		return "", "", false
	}
	return line[:eq], line[eq+1:], true
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
