package lanectl

import (
	"strings"
	"unicode"
)

// SSHAlias is a safe OpenSSH Host name from a customer folder and session.
func SSHAlias(folder, name string) string {
	s := strings.TrimSpace(folder)
	n := strings.TrimSpace(name)
	n = strings.TrimSuffix(n, ".ini")
	if s != "" && n != "" {
		s = s + "-" + n
	} else if n != "" {
		s = n
	}
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "host"
	}
	return out
}

// FolderOfName picks the longest mapped folder that prefixes a session name.
func FolderOfName(name string, folders []string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.ReplaceAll(n, "\\", "/")
	best := ""
	for _, f := range folders {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		fl := strings.ToLower(f)
		if n == fl || strings.HasPrefix(n, fl+"/") || strings.HasPrefix(n, fl+"-") || strings.HasPrefix(n, fl+" ") {
			if len(fl) >= len(best) {
				best = f
			}
		}
	}
	return best
}
