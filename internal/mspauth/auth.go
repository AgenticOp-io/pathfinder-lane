package mspauth

import (
	"context"
	"fmt"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/idp"
)

// Authenticator performs engineer sign-in using enrolled MSP settings.
type Authenticator struct {
	Verifier idp.TokenVerifier
	Home     string
}

// NewAuthenticator builds the default authenticator.
func NewAuthenticator(home string) *Authenticator {
	return &Authenticator{Verifier: idp.DefaultVerifier{}, Home: home}
}

// SignIn runs interactive OIDC for enrolled cloud providers.
func (a *Authenticator) SignIn(ctx context.Context, enroll Enrollment) (UserSession, error) {
	enroll.Provider = enroll.Provider.Normalize()
	if enroll.Provider == ProviderLocal {
		now := time.Now()
		s := UserSession{Provider: ProviderLocal, AuthenticatedAt: now, Name: "local"}
		if err := SaveUserSession(a.Home, s); err != nil {
			return UserSession{}, err
		}
		return s, nil
	}
	claims, refresh, expiresIn, err := idp.LoginInteractive(ctx, enroll.LoginConfig(), 0)
	if err != nil {
		return UserSession{}, err
	}
	if refresh != "" {
		if err := StoreRefreshToken(enroll, refresh); err != nil {
			return UserSession{}, fmt.Errorf("store refresh token: %w", err)
		}
	}
	now := time.Now()
	sess := UserSession{
		Provider:        claims.Provider,
		Subject:         claims.Subject,
		Email:           claims.Email,
		Name:            claims.Name,
		Roles:           claims.Roles,
		AuthenticatedAt: now,
		ExpiresAt:       now.Add(time.Duration(expiresIn) * time.Second),
	}
	if err := SaveUserSession(a.Home, sess); err != nil {
		return UserSession{}, err
	}
	return sess, nil
}

// EnrollAndVerify saves MSP enrollment; cloud providers sign in once to verify.
func (a *Authenticator) EnrollAndVerify(ctx context.Context, enroll Enrollment) (Enrollment, UserSession, error) {
	if err := ValidateEnrollment(enroll); err != nil {
		return Enrollment{}, UserSession{}, err
	}
	enroll.EnrolledAt = time.Now()
	if enroll.Provider.RequiresCloudLogin() {
		sess, err := a.SignIn(ctx, enroll)
		if err != nil {
			return Enrollment{}, UserSession{}, fmt.Errorf("super admin sign-in failed: %w", err)
		}
		enroll.EnrolledBy = firstNonEmpty(sess.Email, sess.Name, sess.Subject)
		if err := SaveEnrollment(enroll); err != nil {
			return Enrollment{}, UserSession{}, err
		}
		return enroll, sess, nil
	}
	if err := SaveEnrollment(enroll); err != nil {
		return Enrollment{}, UserSession{}, err
	}
	sess, err := a.SignIn(ctx, enroll)
	return enroll, sess, err
}

// CurrentSession loads and validates the per-user session.
func (a *Authenticator) CurrentSession(enroll Enrollment) (UserSession, bool, error) {
	sess, found, err := LoadUserSession(a.Home)
	if err != nil {
		return UserSession{}, false, err
	}
	if !found {
		return UserSession{}, false, nil
	}
	if !SessionValid(enroll, sess, time.Now()) {
		return sess, false, nil
	}
	return sess, true, nil
}

// SignOut clears engineer session.
func (a *Authenticator) SignOut(enroll Enrollment) error {
	return ClearUserSession(a.Home, enroll)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
