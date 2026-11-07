package portal

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/gofiber/fiber/v3"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/oauth2"
)

// The OAuth provider is local, but code exchange, PKCE, JWT signatures, issuer,
// audience, nonce and expiry are exercised by the production Google implementation.
type testGoogle struct {
	provider *googleProvider
	mu       sync.Mutex
	grants   map[string]testGrant
}

type testGrant struct {
	subject, email, nonce, challenge string
	alter                            func(map[string]any)
}

func newTestGoogle(t *testing.T) *testGoogle {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-key"))
	if err != nil {
		t.Fatal(err)
	}
	mock := &testGoogle{grants: map[string]testGrant{}}
	var issuer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/keys" {
			_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
				{Key: &key.PublicKey, KeyID: "test-key", Algorithm: "RS256", Use: "sig"},
			}})
			return
		}
		if r.URL.Path != "/token" || r.Method != "POST" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", 400)
			return
		}
		mock.mu.Lock()
		grant, ok := mock.grants[r.Form.Get("code")]
		delete(mock.grants, r.Form.Get("code"))
		mock.mu.Unlock()
		challenge := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
		if !ok || base64.RawURLEncoding.EncodeToString(challenge[:]) != grant.challenge ||
			r.Form.Get("client_id") != "test-client" || r.Form.Get("client_secret") != "test-secret" ||
			r.Form.Get("redirect_uri") != "https://edgenet.example/api/auth/google/callback" {
			http.Error(w, `{"error":"invalid_grant"}`, 400)
			return
		}
		claims := map[string]any{
			"iss": issuer, "aud": "test-client", "sub": grant.subject,
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
			"email": grant.email, "email_verified": true, "name": "Test User", "nonce": grant.nonce,
		}
		if grant.alter != nil {
			grant.alter(claims)
		}
		raw, _ := json.Marshal(claims)
		signed, err := signer.Sign(raw)
		if err != nil {
			http.Error(w, "signing failed", 500)
			return
		}
		idToken, _ := signed.CompactSerialize()
		if claims["testCorruptSignature"] == true {
			parts := strings.Split(idToken, ".")
			signature, _ := base64.RawURLEncoding.DecodeString(parts[2])
			signature[0] ^= 1
			parts[2] = base64.RawURLEncoding.EncodeToString(signature)
			idToken = strings.Join(parts, ".")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "test-access", "token_type": "Bearer", "id_token": idToken})
	}))
	issuer = server.URL
	t.Cleanup(server.Close)
	mock.provider = &googleProvider{
		oauth: oauth2.Config{
			ClientID: "test-client", ClientSecret: "test-secret",
			RedirectURL: "https://edgenet.example/api/auth/google/callback", Scopes: []string{"openid", "email", "profile"},
			Endpoint: oauth2.Endpoint{AuthURL: issuer + "/authorize", TokenURL: issuer + "/token", AuthStyle: oauth2.AuthStyleInParams},
		},
		verifier: oidc.NewVerifier(issuer, oidc.NewRemoteKeySet(context.Background(), issuer+"/keys"), &oidc.Config{ClientID: "test-client"}),
	}
	return mock
}

func (g *testGoogle) grant(subject, email, nonce, challenge string, alter func(map[string]any)) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	code := randomID()
	g.grants[code] = testGrant{subject: subject, email: email, nonce: nonce, challenge: challenge, alter: alter}
	return code
}

func appRequest(t *testing.T, app *fiber.App, path string, cookies ...*http.Cookie) *http.Response {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	res, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

func responseCookie(t *testing.T, response *http.Response, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %s missing", name)
	return nil
}

func oauthSetup(t *testing.T) (*Server, *fiber.App, *testGoogle) {
	t.Helper()
	s, err := Open(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.ConfigureAuth(AuthConfig{AppURL: "https://edgenet.example", GoogleClientID: "test-client", GoogleClientSecret: "test-secret"}); err != nil {
		t.Fatal(err)
	}
	g := newTestGoogle(t)
	s.google = g.provider
	return s, s.App(), g
}

func googleLogin(t *testing.T, app *fiber.App, g *testGoogle, subject, email string) client {
	t.Helper()
	start := appRequest(t, app, "/api/auth/google")
	if start.StatusCode != 302 {
		t.Fatalf("login start: %d", start.StatusCode)
	}
	u, err := url.Parse(start.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	query := u.Query()
	if query.Get("code_challenge_method") != "S256" || query.Get("nonce") == "" || query.Get("scope") != "openid email profile" {
		t.Fatalf("invalid authorization request: %s", u)
	}
	code := g.grant(subject, email, query.Get("nonce"), query.Get("code_challenge"), nil)
	callback := appRequest(t, app, "/api/auth/google/callback?state="+query.Get("state")+"&code="+code, responseCookie(t, start, oauthCookie))
	if callback.StatusCode != 302 || callback.Header.Get("Location") != "https://edgenet.example/" {
		t.Fatalf("callback: %d %s", callback.StatusCode, callback.Header.Get("Location"))
	}
	cookie := responseCookie(t, callback, sessionCookie)
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge != int(sessionLifetime.Seconds()) {
		t.Fatalf("incorrect session cookie policy: %+v", cookie)
	}
	cl := client{t: t, app: app, cookie: cookie, origin: "https://edgenet.example"}
	info := decode[struct {
		CSRFToken string `json:"csrfToken"`
		User      User   `json:"user"`
	}](t, cl.request("GET", "/api/auth/session", nil, 200))
	if info.User.Provider != "google" || info.User.Email != email || info.CSRFToken == "" {
		t.Fatalf("invalid session: %+v", info)
	}
	cl.csrf = info.CSRFToken
	return cl
}

func TestGoogleLoginSessionsAndCSRF(t *testing.T) {
	s, app, g := oauthSetup(t)
	anonymous := client{t: t, app: app}
	info := decode[map[string]any](t, anonymous.request("GET", "/api/auth/session", nil, 200))
	if info["user"] != nil || info["googleEnabled"] != true || info["adminEnabled"] != false {
		t.Fatalf("invalid public session info: %+v", info)
	}
	anonymous.request("GET", "/api/admin/state", nil, 401)
	user := googleLogin(t, app, g, "google-subject-1", "first@example.com")
	input := map[string]string{"name": "ESP32", "kind": "esp32", "target": "esp32"}
	noCSRF := user
	noCSRF.csrf = ""
	noCSRF.request("POST", "/api/admin/devices", input, 403)
	noCSRF.request("POST", "/api/auth/logout", nil, 403)
	wrongOrigin := user
	wrongOrigin.origin = "https://other.example"
	wrongOrigin.request("POST", "/api/admin/devices", input, 403)
	wrongCSRF := user
	wrongCSRF.csrf = "forged"
	wrongCSRF.request("POST", "/api/admin/devices", input, 403)
	d, _ := user.enroll("My sensor", "esp32")
	// Repeat login and email changes retain ownership through the stable subject.
	returning := googleLogin(t, app, g, "google-subject-1", "renamed@example.com")
	st := decode[State](t, returning.request("GET", "/api/admin/state", nil, 200))
	if len(st.Devices) != 1 || st.Devices[0].ID != d.ID {
		t.Fatal("returning account lost its workspace")
	}
	other := googleLogin(t, app, g, "different-subject", "renamed@example.com")
	st = decode[State](t, other.request("GET", "/api/admin/state", nil, 200))
	if len(st.Devices) != 0 {
		t.Fatal("email matching must not merge accounts")
	}
	user.request("POST", "/api/auth/logout", nil, 204)
	user.request("GET", "/api/admin/state", nil, 401)
	returning.request("GET", "/api/admin/state", nil, 200)
	if err := s.db.Update(func(tx *bolt.Tx) error {
		var session browserSession
		if err := getJSON(tx, "sessions", hash(returning.cookie.Value), &session); err != nil {
			return err
		}
		session.ExpiresAt = time.Now().Add(-time.Minute)
		return putJSON(tx, "sessions", hash(returning.cookie.Value), session)
	}); err != nil {
		t.Fatal(err)
	}
	returning.request("GET", "/api/admin/state", nil, 401)
}

func TestGoogleRejectsInvalidTokens(t *testing.T) {
	g := newTestGoogle(t)
	cases := map[string]func(map[string]any){
		"signature":        func(c map[string]any) { c["testCorruptSignature"] = true },
		"audience":         func(c map[string]any) { c["aud"] = "another-client" },
		"issuer":           func(c map[string]any) { c["iss"] = "https://attacker.example" },
		"expiry":           func(c map[string]any) { c["exp"] = time.Now().Add(-time.Hour).Unix() },
		"nonce":            func(c map[string]any) { c["nonce"] = "wrong" },
		"unverified email": func(c map[string]any) { c["email_verified"] = false },
		"missing subject":  func(c map[string]any) { c["sub"] = "" },
	}
	for name, alter := range cases {
		t.Run(name, func(t *testing.T) {
			verifier := oauth2.GenerateVerifier()
			challenge := sha256.Sum256([]byte(verifier))
			code := g.grant("subject", "user@example.com", "expected", base64.RawURLEncoding.EncodeToString(challenge[:]), alter)
			if _, err := g.provider.exchange(context.Background(), code, "expected", verifier); err == nil {
				t.Fatal("invalid identity accepted")
			}
		})
	}
	challenge := sha256.Sum256([]byte("correct-verifier"))
	code := g.grant("subject", "user@example.com", "nonce", base64.RawURLEncoding.EncodeToString(challenge[:]), nil)
	if _, err := g.provider.exchange(context.Background(), code, "nonce", "wrong-verifier"); err == nil {
		t.Fatal("code exchange accepted the wrong PKCE verifier")
	}
}

func TestFailedOAuthCallbackDoesNotCreateSession(t *testing.T) {
	s, app, _ := oauthSetup(t)
	for _, query := range []string{"error=access_denied", "code=invalid-code"} {
		start := appRequest(t, app, "/api/auth/google")
		u, _ := url.Parse(start.Header.Get("Location"))
		callback := appRequest(t, app, "/api/auth/google/callback?state="+u.Query().Get("state")+"&"+query, responseCookie(t, start, oauthCookie))
		if !strings.Contains(callback.Header.Get("Location"), "auth_error=") {
			t.Fatal("failed callback should return a sign-in error")
		}
		for _, cookie := range callback.Cookies() {
			if cookie.Name == sessionCookie && cookie.Value != "" {
				t.Fatal("failed callback created a session cookie")
			}
		}
	}
	if err := s.db.View(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte("sessions")); b != nil && b.Stats().KeyN != 0 {
			t.Fatal("failed callback persisted a session")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSessionReplacementRevokesPreviousCookie(t *testing.T) {
	s, admin := setup(t)
	previous := sessionClient(t, s, admin.app, "alice")
	raw, err := s.createSession(googleIdentity{Subject: "bob", Email: "bob@example.com", Name: "Bob"}, previous.cookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	previous.request("GET", "/api/admin/state", nil, 401)
	user, session, err := s.readSession(raw)
	if err != nil || user.Name != "Bob" || session.CSRFToken == previous.csrf {
		t.Fatal("session replacement failed")
	}
	// Local bearer logout must not revoke an unrelated Google cookie.
	admin.cookie = &http.Cookie{Name: sessionCookie, Value: raw}
	admin.request("POST", "/api/auth/logout", nil, 204)
	if _, _, err := s.readSession(raw); err != nil {
		t.Fatal("local logout revoked the Google account")
	}
}

func TestOAuthStateBindingReplayAndExpiry(t *testing.T) {
	s, app, g := oauthSetup(t)
	start := appRequest(t, app, "/api/auth/google")
	u, _ := url.Parse(start.Header.Get("Location"))
	q := u.Query()
	code := g.grant("subject", "user@example.com", q.Get("nonce"), q.Get("code_challenge"), nil)
	path := "/api/auth/google/callback?state=" + q.Get("state") + "&code=" + code
	for _, cookies := range [][]*http.Cookie{nil, {{Name: oauthCookie, Value: "another-browser"}}} {
		res := appRequest(t, app, path, cookies...)
		if !strings.Contains(res.Header.Get("Location"), "auth_error=expired") {
			t.Fatal("unbound OAuth callback accepted")
		}
	}
	cookie := responseCookie(t, start, oauthCookie)
	res := appRequest(t, app, path, cookie)
	if res.Header.Get("Location") != "https://edgenet.example/" {
		t.Fatal("valid state rejected")
	}
	res = appRequest(t, app, path, cookie)
	if !strings.Contains(res.Header.Get("Location"), "auth_error=expired") {
		t.Fatal("OAuth state replay accepted")
	}
	start = appRequest(t, app, "/api/auth/google")
	u, _ = url.Parse(start.Header.Get("Location"))
	state := u.Query().Get("state")
	if err := s.db.Update(func(tx *bolt.Tx) error {
		var attempt oauthAttempt
		if err := getJSON(tx, "oauthAttempts", hash(state), &attempt); err != nil {
			return err
		}
		attempt.ExpiresAt = time.Now().Add(-time.Minute)
		return putJSON(tx, "oauthAttempts", hash(state), attempt)
	}); err != nil {
		t.Fatal(err)
	}
	res = appRequest(t, app, "/api/auth/google/callback?state="+state+"&code=x", responseCookie(t, start, oauthCookie))
	if !strings.Contains(res.Header.Get("Location"), "auth_error=expired") {
		t.Fatal("expired state accepted")
	}
}

func sessionClient(t *testing.T, s *Server, app *fiber.App, subject string) client {
	t.Helper()
	raw, err := s.createSession(googleIdentity{Subject: subject, Name: subject, Email: subject + "@example.com"}, "")
	if err != nil {
		t.Fatal(err)
	}
	_, session, err := s.readSession(raw)
	if err != nil {
		t.Fatal(err)
	}
	return client{t: t, app: app, cookie: &http.Cookie{Name: sessionCookie, Value: raw}, csrf: session.CSRFToken}
}

func TestWorkspaceIsolation(t *testing.T) {
	s, admin := setup(t)
	a := sessionClient(t, s, admin.app, "alice")
	b := sessionClient(t, s, admin.app, "bob")
	da, agentA := a.enroll("Alice sensor", "esp32")
	db, agentB := b.enroll("Bob sensor", "esp32")
	admin.enroll("Legacy sensor", "esp32")
	fwa := decode[Firmware](t, a.request("POST", "/api/admin/firmware", artifact(t, "1.0", "esp32", "alice firmware"), 201))
	fwb := decode[Firmware](t, b.request("POST", "/api/admin/firmware", artifact(t, "1.0", "esp32", "bob firmware"), 201))
	for _, cl := range []client{b, admin} {
		cl.request("POST", "/api/admin/devices/"+da.ID+"/rotate-token", nil, 404)
		cl.request("DELETE", "/api/admin/devices/"+da.ID, nil, 404)
	}
	batch := func(fw string, ids ...string) map[string]any {
		return map[string]any{"firmwareId": fw, "deviceIds": ids}
	}
	a.request("POST", "/api/admin/deployments", batch(fwb.ID, da.ID), 404)
	a.request("POST", "/api/admin/deployments", batch(fwa.ID, da.ID, db.ID), 404)
	st := decode[State](t, a.request("GET", "/api/admin/state", nil, 200))
	if len(st.Deployments) != 0 {
		t.Fatal("cross-owner batch created partial jobs")
	}
	jobs := decode[[]Deployment](t, a.request("POST", "/api/admin/deployments", batch(fwa.ID, da.ID), 201))
	job := jobs[0]
	b.request("POST", "/api/admin/deployments/"+job.ID+"/cancel", nil, 404)
	agentB.request("POST", "/api/agent/jobs/"+job.ID+"/status", map[string]string{"status": "downloading"}, 404)
	agentB.request("GET", "/api/agent/jobs/"+job.ID+"/artifact", nil, 404)
	if got := decode[[]any](t, agentB.request("GET", "/api/agent/jobs", nil, 200)); len(got) != 0 {
		t.Fatal("another owner's jobs leaked")
	}
	if got := agentA.request("GET", "/api/agent/jobs/"+job.ID+"/artifact", nil, 200); string(got) != "alice firmware" {
		t.Fatal("incorrect artifact")
	}
	agentA.request("POST", "/api/agent/heartbeat", map[string]string{"version": "1.0"}, 200)
	for _, cl := range []client{b, admin} {
		raw := cl.request("GET", "/api/admin/state", nil, 200)
		if bytes.Contains(raw, []byte("Alice")) || bytes.Contains(raw, []byte(da.ID)) || bytes.Contains(raw, []byte(fwa.ID)) || bytes.Contains(raw, []byte(job.ID)) {
			t.Fatal("workspace snapshot leaked another user's data")
		}
	}
	a.request("POST", "/api/admin/deployments/"+job.ID+"/cancel", nil, 204)
	a.request("DELETE", "/api/admin/devices/"+da.ID, nil, 204)
	agentA.request("GET", "/api/agent/jobs", nil, 401)
	agentB.request("GET", "/api/agent/jobs", nil, 200)
}

func TestUserSessionsPersistAndLegacyMigration(t *testing.T) {
	dir := t.TempDir()
	db, err := bolt.Open(filepath.Join(dir, "edgenet.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	legacy := State{Devices: []Device{{ID: "legacy-id", Name: "Existing board", Kind: "esp32"}}, Firmware: []Firmware{}, Deployments: []Deployment{}, Events: []Event{}, TokenHashes: map[string]string{"legacy-id": hash("device-secret")}}
	if err := db.Update(func(tx *bolt.Tx) error { return putJSON(tx, "portal", "state", legacy) }); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir, adminKey)
	if err != nil {
		t.Fatal(err)
	}
	app := s.App()
	admin := client{t: t, app: app, token: adminKey}
	st := decode[State](t, admin.request("GET", "/api/admin/state", nil, 200))
	if len(st.Devices) != 1 || st.Devices[0].ID != "legacy-id" {
		t.Fatal("migration lost existing devices")
	}
	agent := client{t: t, app: app, id: "legacy-id", token: "device-secret"}
	agent.request("GET", "/api/agent/jobs", nil, 200)
	user := sessionClient(t, s, app, "new-user")
	st = decode[State](t, user.request("GET", "/api/admin/state", nil, 200))
	if len(st.Devices) != 0 {
		t.Fatal("new user inherited legacy devices")
	}
	d, _ := user.enroll("Personal board", "esp32")
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir, adminKey)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	user.app = s.App()
	st = decode[State](t, user.request("GET", "/api/admin/state", nil, 200))
	if len(st.Devices) != 1 || st.Devices[0].ID != d.ID {
		t.Fatal("session or user workspace did not persist")
	}
	if err := s.db.View(func(tx *bolt.Tx) error {
		if raw := tx.Bucket([]byte("sessions")).Get([]byte(user.cookie.Value)); raw != nil {
			t.Fatal("session IDs must be hashed at rest")
		}
		if raw := tx.Bucket([]byte("portal")).Get([]byte("state")); raw != nil {
			t.Fatal("legacy state was not migrated")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestAuthConfiguration(t *testing.T) {
	s, admin := setup(t)
	for _, origin := range []string{"", "https://example.com/path", "https://example.com?x=1", "http://example.com", "//example.com", "https://user:secret@example.com"} {
		if err := s.ConfigureAuth(AuthConfig{AppURL: origin}); err == nil {
			t.Fatalf("invalid origin accepted: %s", origin)
		}
	}
	if err := s.ConfigureAuth(AuthConfig{AppURL: "https://example.com", GoogleClientID: "missing-secret"}); err == nil {
		t.Fatal("partial Google configuration accepted")
	}
	if err := s.ConfigureAuth(AuthConfig{AppURL: "http://localhost:5173"}); err != nil {
		t.Fatal(err)
	}
	anonymous := client{t: t, app: admin.app}
	anonymous.request("GET", "/api/auth/google", nil, 503)
	res := appRequest(t, admin.app, "/api/auth/session")
	if res.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("session response must not be cached")
	}
	_, _ = io.Copy(io.Discard, res.Body)
}
