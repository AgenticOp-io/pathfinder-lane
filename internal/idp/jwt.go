package idp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// rawClaims are JWT payload fields from an id_token.
type rawClaims struct {
	Subject string   `json:"sub"`
	Email   string   `json:"email"`
	UPN     string   `json:"upn"`
	Name    string   `json:"name"`
	HD      string   `json:"hd"`
	Tenant  string   `json:"tid"`
	Roles   []string `json:"roles"`
}

// TokenVerifier validates an id_token signature (JWKS in production).
type TokenVerifier interface {
	Verify(raw string, cfg LoginConfig) (rawClaims, error)
}

// DefaultVerifier parses claims without signature verification.
type DefaultVerifier struct{}

func (DefaultVerifier) Verify(raw string, cfg LoginConfig) (rawClaims, error) {
	return ParseIDTokenClaims(raw)
}

func ParseIDTokenClaims(raw string) (rawClaims, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return rawClaims{}, fmt.Errorf("empty id_token")
	}
	parts := strings.Split(raw, ".")
	if len(parts) < 2 {
		return rawClaims{}, fmt.Errorf("id_token is not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return rawClaims{}, fmt.Errorf("decode id_token payload: %w", err)
	}
	var c rawClaims
	if err := json.Unmarshal(payload, &c); err != nil {
		return rawClaims{}, fmt.Errorf("parse id_token claims: %w", err)
	}
	if c.Subject == "" {
		return rawClaims{}, fmt.Errorf("id_token missing sub claim")
	}
	return c, nil
}

func claimsMatchConfig(cfg LoginConfig, c rawClaims) error {
	domain := strings.TrimSpace(cfg.Domain)
	if domain == "" {
		return nil
	}
	domain = strings.TrimPrefix(strings.ToLower(domain), "@")
	switch cfg.Provider {
	case ProviderGoogle:
		hd := strings.ToLower(strings.TrimSpace(c.HD))
		email := strings.ToLower(strings.TrimSpace(c.Email))
		if hd == domain || strings.HasSuffix(email, "@"+domain) {
			return nil
		}
		return fmt.Errorf("Google account is not on domain %s", cfg.Domain)
	case ProviderEntra:
		email := strings.ToLower(firstNonEmpty(c.Email, c.UPN))
		if cfg.TenantID != "" && c.Tenant != "" && !strings.EqualFold(c.Tenant, cfg.TenantID) {
			return fmt.Errorf("token tenant %s does not match enrollment", c.Tenant)
		}
		if email == "" || !strings.HasSuffix(email, "@"+domain) {
			return fmt.Errorf("Microsoft account is not on domain %s", cfg.Domain)
		}
	}
	return nil
}

// LoginConfigFromEnrollment maps MSP enrollment to idp login config.
func LoginConfigFromEnrollment(provider Provider, tenantID, clientID, domain, authority string) LoginConfig {
	return LoginConfig{
		Provider:  provider.Normalize(),
		TenantID:  tenantID,
		ClientID:  clientID,
		Domain:    domain,
		Authority: authority,
	}
}
