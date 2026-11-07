package portal

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/oauth2"
)

const sessionCookie = "edgenet_session"
const oauthCookie = "edgenet_oauth"

func workspaceID(c fiber.Ctx) string { return c.Locals("workspaceID").(string) }

func (s *Server) authRoutes(app *fiber.App) {
	app.Use("/api/auth", func(c fiber.Ctx) error { c.Set("Referrer-Policy", "no-referrer"); return c.Next() })
	app.Get("/api/auth/session", s.sessionInfo)
	app.Post("/api/auth/logout", s.requireUser, s.logout)
	// Bound unauthenticated login attempts without limiting device traffic.
	app.Get("/api/auth/google", limiter.New(limiter.Config{Max: 20, Expiration: time.Minute}), s.startGoogle)
	app.Get("/api/auth/google/callback", s.finishGoogle)
}

func (s *Server) authenticate(c fiber.Ctx) (User, string, error) {
	if c.Get("Authorization") != "" {
		if s.adminToken == "" || !strings.HasPrefix(c.Get("Authorization"), "Bearer ") || !equal(hash(bearer(c)), hash(s.adminToken)) {
			return User{}, "", fiber.ErrUnauthorized
		}
		return User{ID: localWorkspace, Name: "Local administrator", Provider: "admin"}, "", nil
	}
	user, session, err := s.readSession(c.Cookies(sessionCookie))
	return user, session.CSRFToken, err
}

func (s *Server) requireUser(c fiber.Ctx) error {
	user, csrf, err := s.authenticate(c)
	if err != nil {
		return err
	}
	if user.Provider != "admin" && c.Method() != "GET" && c.Method() != "HEAD" {
		if csrf == "" || !equal(c.Get("X-CSRF-Token"), csrf) {
			return fiber.NewError(403, "Invalid CSRF token; refresh and try again")
		}
		if origin := c.Get("Origin"); origin != "" && origin != s.auth.AppURL {
			return fiber.NewError(403, "Invalid request origin")
		}
	}
	c.Locals("workspaceID", user.ID)
	c.Locals("user", user)
	c.Locals("csrfToken", csrf)
	return c.Next()
}

func (s *Server) sessionInfo(c fiber.Ctx) error {
	user, csrf, err := s.authenticate(c)
	var profile *User
	if err == nil {
		profile = &user
	} else if !errors.Is(err, fiber.ErrUnauthorized) {
		return err
	} else {
		csrf = ""
	}
	return c.JSON(fiber.Map{
		"user": profile, "csrfToken": csrf,
		"googleEnabled": s.google != nil, "adminEnabled": s.adminToken != "",
	})
}

func (s *Server) cookie(c fiber.Ctx, name, value string, lifetime time.Duration) {
	maxAge := int(lifetime.Seconds())
	if lifetime < 0 {
		maxAge = -1
	}
	c.Cookie(&fiber.Cookie{
		Name: name, Value: value, Path: "/", HTTPOnly: true,
		Secure: strings.HasPrefix(s.auth.AppURL, "https://"), SameSite: "Lax",
		MaxAge: maxAge, Expires: time.Now().Add(lifetime),
	})
}

func (s *Server) logout(c fiber.Ctx) error {
	// A local bearer login has no server session and must not revoke a different
	// Google login that happens to share this browser's cookie jar.
	if c.Get("Authorization") != "" {
		return c.SendStatus(204)
	}
	if raw := c.Cookies(sessionCookie); raw != "" {
		if err := s.db.Update(func(tx *bolt.Tx) error { return deleteKey(tx, "sessions", hash(raw)) }); err != nil {
			return err
		}
	}
	s.cookie(c, sessionCookie, "", -time.Hour)
	return c.SendStatus(204)
}

func (s *Server) startGoogle(c fiber.Ctx) error {
	if s.google == nil {
		return fiber.NewError(503, "Google sign-in is not configured")
	}
	state, browser := randomID(), randomID()
	attempt := oauthAttempt{
		BrowserHash: hash(browser), Nonce: randomID(), Verifier: oauth2.GenerateVerifier(),
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
	}
	if err := s.db.Update(func(tx *bolt.Tx) error {
		if err := pruneExpired(tx, "oauthAttempts"); err != nil {
			return err
		}
		return putJSON(tx, "oauthAttempts", hash(state), attempt)
	}); err != nil {
		return err
	}
	s.cookie(c, oauthCookie, browser, 10*time.Minute)
	return c.Redirect().Status(302).To(s.google.authorize(state, attempt.Nonce, attempt.Verifier))
}

func (s *Server) finishGoogle(c fiber.Ctx) error {
	if s.google == nil {
		return fiber.NewError(503, "Google sign-in is not configured")
	}
	state, browser := c.Query("state"), c.Cookies(oauthCookie)
	s.cookie(c, oauthCookie, "", -time.Hour)
	var attempt oauthAttempt
	err := s.db.Update(func(tx *bolt.Tx) error {
		if state == "" || browser == "" {
			return fiber.ErrUnauthorized
		}
		if err := getJSON(tx, "oauthAttempts", hash(state), &attempt); err != nil {
			return err
		}
		if !time.Now().Before(attempt.ExpiresAt) || !equal(attempt.BrowserHash, hash(browser)) {
			return fiber.ErrUnauthorized
		}
		return deleteKey(tx, "oauthAttempts", hash(state))
	})
	if err != nil {
		if !errors.Is(err, fiber.ErrUnauthorized) {
			return err
		}
		return c.Redirect().Status(302).To(s.auth.AppURL + "/?auth_error=expired")
	}
	if c.Query("error") != "" || c.Query("code") == "" {
		return c.Redirect().Status(302).To(s.auth.AppURL + "/?auth_error=cancelled")
	}
	ctx, cancel := context.WithTimeout(c.Context(), 15*time.Second)
	defer cancel()
	identity, err := s.google.exchange(ctx, c.Query("code"), attempt.Nonce, attempt.Verifier)
	if err != nil {
		// Provider errors may contain tokens; return only a fixed, public message.
		return c.Redirect().Status(302).To(s.auth.AppURL + "/?auth_error=failed")
	}
	raw, err := s.createSession(identity, c.Cookies(sessionCookie))
	if err != nil {
		return err
	}
	s.cookie(c, sessionCookie, raw, sessionLifetime)
	c.Set("Referrer-Policy", "no-referrer")
	return c.Redirect().Status(302).To(s.auth.AppURL + "/")
}
