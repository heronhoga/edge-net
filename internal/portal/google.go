package portal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthConfig struct {
	AppURL             string
	GoogleClientID     string
	GoogleClientSecret string
}

type googleIdentity struct {
	Subject string
	Email   string
	Name    string
}

type identityProvider interface {
	authorize(state, nonce, verifier string) string
	exchange(ctx context.Context, code, nonce, verifier string) (googleIdentity, error)
}

type googleProvider struct {
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// ConfigureAuth is called before serving requests. Only the configured public origin
// is used for redirects and cookie policy; forwarded request headers are not trusted.
func (s *Server) ConfigureAuth(config AuthConfig) error {
	config.AppURL = strings.TrimRight(config.AppURL, "/")
	u, err := url.Parse(config.AppURL)
	if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("APP_URL must be an absolute origin, such as https://edgenet.example.com")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")) {
		return errors.New("APP_URL must use HTTPS, except for localhost development")
	}
	if (config.GoogleClientID == "") != (config.GoogleClientSecret == "") {
		return errors.New("set both GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET")
	}
	s.auth = config
	if config.GoogleClientID == "" {
		s.google = nil
		return nil
	}
	s.google = &googleProvider{
		oauth: oauth2.Config{
			ClientID: config.GoogleClientID, ClientSecret: config.GoogleClientSecret,
			RedirectURL: config.AppURL + "/api/auth/google/callback",
			Endpoint:    google.Endpoint, Scopes: []string{oidc.ScopeOpenID, "email", "profile"},
		},
		verifier: oidc.NewVerifier("https://accounts.google.com",
			oidc.NewRemoteKeySet(oidc.ClientContext(context.Background(), &http.Client{Timeout: 15 * time.Second}), "https://www.googleapis.com/oauth2/v3/certs"),
			&oidc.Config{ClientID: config.GoogleClientID}),
	}
	return nil
}

func (p *googleProvider) authorize(state, nonce, verifier string) string {
	return p.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("prompt", "select_account"))
}

func (p *googleProvider) exchange(ctx context.Context, code, nonce, verifier string) (googleIdentity, error) {
	token, err := p.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return googleIdentity{}, fmt.Errorf("exchange authorization code: %w", err)
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return googleIdentity{}, errors.New("missing ID token")
	}
	id, err := p.verifier.Verify(ctx, raw)
	if err != nil {
		return googleIdentity{}, fmt.Errorf("verify ID token: %w", err)
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Nonce         string `json:"nonce"`
	}
	if err := id.Claims(&claims); err != nil {
		return googleIdentity{}, err
	}
	if !equal(claims.Nonce, nonce) || !claims.EmailVerified || !validText(id.Subject, 255) || !validText(claims.Email, 320) {
		return googleIdentity{}, errors.New("invalid Google identity or nonce")
	}
	name := strings.TrimSpace(claims.Name)
	if !validText(name, 200) {
		name = claims.Email
	}
	return googleIdentity{Subject: id.Subject, Email: claims.Email, Name: name}, nil
}
