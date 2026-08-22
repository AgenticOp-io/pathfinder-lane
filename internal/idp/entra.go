package idp

import "strings"

func entraEndpoints(cfg LoginConfig) (authURL, tokenURL string, scopes []string) {
	tenant := strings.TrimSpace(cfg.TenantID)
	base := strings.TrimRight(strings.TrimSpace(cfg.Authority), "/")
	if base == "" {
		base = "https://login.microsoftonline.com/" + tenant
	}
	return base + "/oauth2/v2.0/authorize", base + "/oauth2/v2.0/token",
		[]string{"openid", "profile", "email", "offline_access"}
}

func googleEndpoints() (authURL, tokenURL string, scopes []string) {
	return "https://accounts.google.com/o/oauth2/v2/auth",
		"https://oauth2.googleapis.com/token",
		[]string{"openid", "profile", "email"}
}
