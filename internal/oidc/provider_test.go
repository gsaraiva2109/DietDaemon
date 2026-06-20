package oidc

import (
	"testing"
)

func TestBuildRegistry(t *testing.T) {
	// Build from configs — no network.
	configs := []ProviderConfig{
		{ID: "google", Name: "Google", Issuer: "https://accounts.google.com", ClientID: "g-id", ClientSecret: "g-secret", RedirectURL: "http://localhost/api/v1/auth/oidc/google/callback"},
		{ID: "authentik", Name: "Authentik", Issuer: "https://auth.example.com", ClientID: "a-id", ClientSecret: "a-secret", RedirectURL: "http://localhost/api/v1/auth/oidc/authentik/callback"},
	}

	reg := BuildRegistry(configs)
	if len(reg) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(reg))
	}

	g, ok := reg["google"]
	if !ok {
		t.Fatal("missing google provider")
	}
	if g.ID != "google" || g.Name != "Google" || g.Issuer != "https://accounts.google.com" {
		t.Fatalf("google config mismatch: %+v", g)
	}
	if len(g.Scopes) != 3 {
		t.Fatalf("expected 3 default scopes, got %d", len(g.Scopes))
	}

	a, ok := reg["authentik"]
	if !ok {
		t.Fatal("missing authentik provider")
	}
	if a.ID != "authentik" {
		t.Fatalf("authentik id mismatch: %s", a.ID)
	}

	// Empty configs → nil map.
	reg2 := BuildRegistry(nil)
	if reg2 != nil {
		t.Fatal("expected nil registry for empty configs")
	}
	reg3 := BuildRegistry([]ProviderConfig{})
	if reg3 != nil {
		t.Fatal("expected nil registry for zero-length configs")
	}
}

func TestBuildRegistryCustomScopes(t *testing.T) {
	configs := []ProviderConfig{
		{ID: "dex", Name: "Dex", Issuer: "https://dex.example.com", ClientID: "d-id", ClientSecret: "d-secret", RedirectURL: "http://localhost/api/v1/auth/oidc/dex/callback", Scopes: []string{"openid", "profile"}},
	}
	reg := BuildRegistry(configs)
	d := reg["dex"]
	if len(d.Scopes) != 2 {
		t.Fatalf("expected 2 custom scopes, got %d", len(d.Scopes))
	}
}

func TestProviderEnsureNoNetwork(t *testing.T) {
	// ensure() should fail without network (bad issuer), but not panic.
	p := &Provider{
		ID: "test", Name: "Test", Issuer: "https://invalid.example.com",
		ClientID: "x", ClientSecret: "x", RedirectURL: "http://localhost/cb",
		Scopes: []string{"openid"},
	}
	// First call: should fail (no network discovery possible).
	err := p.ensure(t.Context())
	if err == nil {
		t.Fatal("expected error for unreachable issuer, got nil")
	}

	// Second call: the once was reset, should re-try and fail again.
	err = p.ensure(t.Context())
	if err == nil {
		t.Fatal("expected error on retry for unreachable issuer, got nil")
	}
}
