package idp

// LoginConfig is provider credentials for an interactive OIDC sign-in.
type LoginConfig struct {
	Provider  Provider
	TenantID  string
	ClientID  string
	Domain    string
	Authority string
}

// SessionClaims are identity fields extracted from an id_token.
type SessionClaims struct {
	Provider Provider
	Subject  string
	Email    string
	Name     string
	Roles    []string
}
