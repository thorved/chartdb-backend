package config

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

var (
	OIDCEnabled      bool
	OIDCIssuerURL    string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string
	OIDCScopes       []string

	OIDCProvider *oidc.Provider
	OAuth2Config *oauth2.Config
	OIDCVerifier *oidc.IDTokenVerifier
)

func InitOIDC() error {
	envValue := os.Getenv("OIDC_ENABLED")
	OIDCEnabled = strings.TrimSpace(envValue) == "true"

	if !OIDCEnabled {
		return nil
	}

	OIDCIssuerURL = os.Getenv("OIDC_ISSUER_URL")
	OIDCClientID = os.Getenv("OIDC_CLIENT_ID")
	OIDCClientSecret = os.Getenv("OIDC_CLIENT_SECRET")
	OIDCRedirectURL = os.Getenv("OIDC_REDIRECT_URL")

	scopesStr := os.Getenv("OIDC_SCOPES")
	if scopesStr == "" {
		OIDCScopes = []string{oidc.ScopeOpenID, "profile", "email"}
	} else {
		OIDCScopes = strings.Split(scopesStr, ",")
	}

	// Validate required config
	if OIDCIssuerURL == "" || OIDCClientID == "" || OIDCClientSecret == "" {
		return fmt.Errorf("OIDC is enabled but missing required configuration")
	}

	// Initialize OIDC provider
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, OIDCIssuerURL)
	if err != nil {
		return fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	OIDCProvider = provider

	// Configure OAuth2
	OAuth2Config = &oauth2.Config{
		ClientID:     OIDCClientID,
		ClientSecret: OIDCClientSecret,
		RedirectURL:  OIDCRedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       OIDCScopes,
	}

	// Configure ID token verifier
	oidcConfig := &oidc.Config{
		ClientID: OIDCClientID,
	}
	OIDCVerifier = provider.Verifier(oidcConfig)

	return nil
}

// GenerateState generates a random state parameter for OAuth2 flow
func GenerateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
