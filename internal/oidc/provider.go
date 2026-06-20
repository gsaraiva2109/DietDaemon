// Package oidc implements OIDC provider discovery, authorization-code flow
// with PKCE + nonce, and ID-token verification. Provider discovery is lazy
// (sync.Once per provider) so the app boots even when a provider is offline.
// Password login is never blocked by an unreachable OIDC provider.
package oidc

import (
	"context"
	"fmt"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// ProviderConfig is the static configuration for one OIDC provider, parsed from
// environment variables. All fields are validated at boot (config.go fail-fast).
type ProviderConfig struct {
	ID           string   // machine key, e.g. "google"
	Name         string   // display name, e.g. "Google"
	Issuer       string   // discovery URL
	ClientID     string   // OAuth2 client id
	ClientSecret string   // OAuth2 client secret
	RedirectURL  string   // {PUBLIC_BASE_URL}/api/v1/auth/oidc/{id}/callback
	Scopes       []string // default ["openid","email","profile"]
	TrustEmail   bool     // trust provider email even without email_verified
}

// Provider wraps a configured OIDC provider with lazy discovery. It is safe
// for concurrent use — discovery runs once and is cached.
type Provider struct {
	ID           string
	Name         string
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	TrustEmail   bool

	once sync.Once
	mu   sync.Mutex
	init *initResult // non-nil after first ensure
}

type initResult struct {
	provider *oidc.Provider
	oauth2   *oauth2.Config
	verifier *oidc.IDTokenVerifier
	err      error
}

// BuildRegistry creates a Provider for each config entry. No network calls are
// made — discovery happens lazily on first use (AuthCodeURL / Exchange /
// VerifyIDToken). The returned map is keyed by provider ID.
func BuildRegistry(configs []ProviderConfig) map[string]*Provider {
	if len(configs) == 0 {
		return nil
	}
	m := make(map[string]*Provider, len(configs))
	for i := range configs {
		c := configs[i]
		scopes := c.Scopes
		if len(scopes) == 0 {
			scopes = []string{oidc.ScopeOpenID, "email", "profile"}
		}
		m[c.ID] = &Provider{
			ID:           c.ID,
			Name:         c.Name,
			Issuer:       c.Issuer,
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			RedirectURL:  c.RedirectURL,
			Scopes:       scopes,
			TrustEmail:   c.TrustEmail,
		}
	}
	return m
}

// ensure runs provider discovery (oidc.NewProvider) and builds the oauth2.Config
// + IDTokenVerifier. It is called transparently by the flow methods. On error,
// the once is reset so a transient provider outage self-heals on the next retry.
func (p *Provider) ensure(ctx context.Context) error {
	p.mu.Lock()
	// Fast path: already initialized.
	if p.init != nil {
		defer p.mu.Unlock()
		return p.init.err
	}

	var init initResult
	p.once.Do(func() {
		provider, err := oidc.NewProvider(ctx, p.Issuer)
		if err != nil {
			init.err = fmt.Errorf("oidc: discover %s (%s): %w", p.ID, p.Issuer, err)
			return
		}
		init.provider = provider
		// Pin the client-auth style to client_secret_post. Left as
		// AuthStyleAutoDetect, oauth2 may send the first token request with one
		// style, and on an unrecognized response retry with the other — but the
		// authorization code is single-use, so the retry fails with
		// "invalid_grant". Authentik (and every standard OIDC provider) accepts
		// the secret in the POST body, so force it and exchange exactly once.
		endpoint := provider.Endpoint()
		endpoint.AuthStyle = oauth2.AuthStyleInParams
		init.oauth2 = &oauth2.Config{
			ClientID:     p.ClientID,
			ClientSecret: p.ClientSecret,
			RedirectURL:  p.RedirectURL,
			Endpoint:     endpoint,
			Scopes:       p.Scopes,
		}
		init.verifier = provider.Verifier(&oidc.Config{ClientID: p.ClientID})
	})

	if init.err != nil {
		// Reset the once so a transient failure doesn't break the provider
		// until restart.
		p.once = sync.Once{}
		p.mu.Unlock()
		return init.err
	}

	p.init = &init
	p.mu.Unlock()
	return nil
}

// AuthCodeURL builds the OAuth2 authorization URL with PKCE (S256), nonce, and
// state. It lazily discovers the provider on first call. pkceVerifier is the
// raw verifier (from oauth2.GenerateVerifier) — S256ChallengeOption derives the
// code_challenge from it, so callers must NOT pre-hash it (double hashing
// produces a challenge the verifier can't satisfy → invalid_grant at exchange).
func (p *Provider) AuthCodeURL(ctx context.Context, state, nonce, pkceVerifier string) (string, error) {
	if err := p.ensure(ctx); err != nil {
		return "", err
	}
	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.S256ChallengeOption(pkceVerifier),
	}
	return p.init.oauth2.AuthCodeURL(state, opts...), nil
}

// Exchange trades the authorization code for an OAuth2 token using PKCE.
// It lazily discovers the provider on first call.
func (p *Provider) Exchange(ctx context.Context, code, pkceVerifier string) (*oauth2.Token, error) {
	if err := p.ensure(ctx); err != nil {
		return nil, err
	}
	return p.init.oauth2.Exchange(ctx, code, oauth2.VerifierOption(pkceVerifier))
}

// IDTokenClaims holds the claims extracted from a verified ID token.
type IDTokenClaims struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
}

// VerifyIDToken validates the raw ID token string via go-oidc, checks the nonce,
// and returns the standard claims. It lazily discovers the provider on first
// call.
func (p *Provider) VerifyIDToken(ctx context.Context, rawIDToken, nonce string) (IDTokenClaims, error) {
	if err := p.ensure(ctx); err != nil {
		return IDTokenClaims{}, err
	}

	idToken, err := p.init.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return IDTokenClaims{}, fmt.Errorf("oidc: verify id_token: %w", err)
	}

	if idToken.Nonce != nonce {
		return IDTokenClaims{}, fmt.Errorf("oidc: nonce mismatch")
	}

	var claims struct {
		Subject       string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return IDTokenClaims{}, fmt.Errorf("oidc: unmarshal claims: %w", err)
	}

	return IDTokenClaims{
		Subject:       claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
	}, nil
}

// UserInfo fetches the provider's UserInfo endpoint. Some providers (notably
// Authentik) return email/profile claims only here, not in the ID token, so the
// callback backfills from UserInfo when the ID token lacks a verified email.
func (p *Provider) UserInfo(ctx context.Context, tok *oauth2.Token) (IDTokenClaims, error) {
	if err := p.ensure(ctx); err != nil {
		return IDTokenClaims{}, err
	}

	ui, err := p.init.provider.UserInfo(ctx, oauth2.StaticTokenSource(tok))
	if err != nil {
		return IDTokenClaims{}, fmt.Errorf("oidc: userinfo: %w", err)
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := ui.Claims(&claims); err != nil {
		return IDTokenClaims{}, fmt.Errorf("oidc: unmarshal userinfo claims: %w", err)
	}

	return IDTokenClaims{
		Subject:       ui.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
	}, nil
}
