package portal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

const adminKey = "integration-test-admin-secret"

type client struct {
	t      *testing.T
	app    *fiber.App
	token  string
	id     string
	cookie *http.Cookie
	csrf   string
	origin string
}

func (cl client) request(method, path string, body any, want int) []byte {
	cl.t.Helper()
	var reader io.Reader
	contentType := "application/json"
	if multipartBody, ok := body.(uploadBody); ok {
		reader, contentType = bytes.NewReader(multipartBody.data), multipartBody.contentType
	} else if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			cl.t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", contentType)
	if cl.token != "" {
		req.Header.Set("Authorization", "Bearer "+cl.token)
	}
	if cl.id != "" {
		req.Header.Set("X-Device-ID", cl.id)
	}
	if cl.cookie != nil {
		req.AddCookie(cl.cookie)
	}
	if cl.csrf != "" {
		req.Header.Set("X-CSRF-Token", cl.csrf)
	}
	if cl.origin != "" {
		req.Header.Set("Origin", cl.origin)
	}
	res, err := cl.app.Test(req)
	if err != nil {
		cl.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		cl.t.Fatal(err)
	}
	if res.StatusCode != want {
		cl.t.Fatalf("%s %s: got %d, want %d: %s", method, path, res.StatusCode, want, raw)
	}
	return raw
}

func decode[T any](t *testing.T, raw []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func setup(t *testing.T) (*Server, client) {
	t.Helper()
	s, err := Open(t.TempDir(), adminKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, client{t: t, app: s.App(), token: adminKey}
}

func (cl client) enroll(name, target string) (Device, client) {
	cl.t.Helper()
	raw := cl.request(http.MethodPost, "/api/admin/devices", map[string]string{"name": name, "kind": "esp32", "target": target, "location": "Lab"}, 201)
	result := decode[struct {
		Device Device `json:"device"`
		Token  string `json:"token"`
	}](cl.t, raw)
	return result.Device, client{t: cl.t, app: cl.app, token: result.Token, id: result.Device.ID}
}

type uploadBody struct {
	data        []byte
	contentType string
}

func artifact(t *testing.T, version, target, content string) uploadBody {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for key, value := range map[string]string{"version": version, "target": target} {
		if err := w.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	f, err := w.CreateFormFile("file", "firmware.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(f, content); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return uploadBody{body.Bytes(), w.FormDataContentType()}
}

func TestDeviceEnrollmentTelemetryAndTokenRotation(t *testing.T) {
	s, admin := setup(t)
	anonymous := client{t: t, app: admin.app}
	anonymous.request("GET", "/api/health", nil, 200)
	anonymous.request("GET", "/api/admin/state", nil, 401)
	anonymous.request("POST", "/api/agent/heartbeat", map[string]string{"version": "1"}, 401)
	admin.request("POST", "/api/admin/devices", map[string]string{"name": " ", "kind": "esp32", "target": "esp32"}, 400)
	for _, kind := range []string{"raspberry-pi", "mini-pc"} {
		admin.request("POST", "/api/admin/devices", map[string]string{"name": "Unsupported", "kind": kind, "target": "unsupported-target"}, 400)
	}
	d, agent := admin.enroll("Sensor", "esp32-v1")
	agent.request("GET", "/api/admin/state", nil, 401)
	initial := decode[State](t, admin.request("GET", "/api/admin/state", nil, 200))
	if initial.Devices[0].Status != "pending" {
		t.Fatal("New device should be pending")
	}
	agent.request("POST", "/api/agent/heartbeat", map[string]any{"version": "1.0.0", "metrics": map[string]any{"cpu": 120}}, 400)
	agent.request("POST", "/api/agent/heartbeat", map[string]any{"version": "1.0.0", "metrics": map[string]any{"cpu": 42, "memory": 25, "uptime": 123}}, 200)
	raw := admin.request("GET", "/api/admin/state", nil, 200)
	if bytes.Contains(raw, []byte(agent.token)) || bytes.Contains(raw, []byte("tokenHashes")) {
		t.Fatal("Snapshot leaks credentials")
	}
	st := decode[State](t, raw)
	if st.Devices[0].Status != "online" || *st.Devices[0].Metrics.CPU != 42 || len(st.Devices[0].History) != 1 {
		t.Fatalf("Bad telemetry: %+v", st.Devices[0])
	}
	if err := s.change(func(st *State) error {
		past := time.Now().Add(-91 * time.Second)
		st.Devices[0].LastSeen = &past
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	st = decode[State](t, admin.request("GET", "/api/admin/state", nil, 200))
	if st.Devices[0].Status != "offline" {
		t.Fatal("Stale heartbeat should be offline")
	}
	rotated := decode[map[string]string](t, admin.request("POST", "/api/admin/devices/"+d.ID+"/rotate-token", nil, 200))
	agent.request("GET", "/api/agent/jobs", nil, 401)
	agent.token = rotated["token"]
	for i := 0; i < 65; i++ {
		agent.request("POST", "/api/agent/heartbeat", map[string]any{"version": "1.0.0", "metrics": map[string]any{"uptime": i}}, 200)
	}
	st = decode[State](t, admin.request("GET", "/api/admin/state", nil, 200))
	if len(st.Devices[0].History) != 60 {
		t.Fatal("History should be bounded at 60")
	}
	if st.Devices[0].Metrics.CPU != nil {
		t.Fatal("Unsupported metric should stay absent")
	}
	admin.request("DELETE", "/api/admin/devices/"+d.ID, nil, 204)
	agent.request("GET", "/api/agent/jobs", nil, 401)
}

func TestFirmwareDeploymentEndToEnd(t *testing.T) {
	_, admin := setup(t)
	d, agent := admin.enroll("ESP32", "esp32-v1")
	other, otherAgent := admin.enroll("Other", "esp32-v1")
	incompatible, _ := admin.enroll("Other target", "esp32-s3")
	content := "test firmware binary contents"
	fw := decode[Firmware](t, admin.request("POST", "/api/admin/firmware", artifact(t, "2.0.0", "esp32-v1", content), 201))
	if fw.SHA256 != hash(content) || fw.Size != int64(len(content)) {
		t.Fatal("Artifact checksum or size is wrong")
	}
	admin.request("POST", "/api/admin/firmware", artifact(t, "2.0.0", "esp32-v1", "different"), 409)
	admin.request("POST", "/api/admin/firmware", artifact(t, "empty", "esp32-v1", ""), 400)
	body := func(ids ...string) map[string]any { return map[string]any{"firmwareId": fw.ID, "deviceIds": ids} }
	admin.request("POST", "/api/admin/deployments", body(d.ID, incompatible.ID), 409)
	admin.request("POST", "/api/admin/deployments", body(d.ID, d.ID), 400)
	st := decode[State](t, admin.request("GET", "/api/admin/state", nil, 200))
	if len(st.Deployments) != 0 {
		t.Fatal("Invalid batch must not create partial jobs")
	}
	jobs := decode[[]Deployment](t, admin.request("POST", "/api/admin/deployments", body(d.ID), 201))
	job := jobs[0]
	admin.request("POST", "/api/admin/deployments", body(d.ID), 409)
	admin.request("DELETE", "/api/admin/devices/"+d.ID, nil, 409)
	polled := decode[[]map[string]json.RawMessage](t, agent.request("GET", "/api/agent/jobs", nil, 200))
	if len(polled) != 1 {
		t.Fatal("Expected one pending job")
	}
	if got := decode[[]any](t, otherAgent.request("GET", "/api/agent/jobs", nil, 200)); len(got) != 0 {
		t.Fatal("Another device can see this job")
	}
	path := "/api/agent/jobs/" + job.ID
	otherAgent.request("GET", path+"/artifact", nil, 404)
	otherAgent.request("POST", path+"/status", map[string]string{"status": "downloading"}, 404)
	agent.request("POST", path+"/status", map[string]string{"status": "succeeded"}, 409)
	agent.request("POST", path+"/status", map[string]string{"status": "downloading"}, 204)
	admin.request("POST", "/api/admin/deployments/"+job.ID+"/cancel", nil, 409)
	if got := agent.request("GET", path+"/artifact", nil, 200); string(got) != content {
		t.Fatalf("Download mismatch: %q", got)
	}
	for _, status := range []string{"downloading", "installing", "succeeded", "succeeded"} {
		agent.request("POST", path+"/status", map[string]string{"status": status}, 204)
	}
	agent.request("POST", path+"/status", map[string]string{"status": "failed"}, 409)
	agent.request("GET", path+"/artifact", nil, 404)
	st = decode[State](t, admin.request("GET", "/api/admin/state", nil, 200))
	if st.Devices[0].Version != "2.0.0" || st.Deployments[0].Status != "succeeded" {
		t.Fatal("Successful deployment did not update state")
	}
	// Offline/pending devices can be queued, and cancellation prevents later claiming.
	jobs = decode[[]Deployment](t, admin.request("POST", "/api/admin/deployments", body(other.ID), 201))
	admin.request("POST", "/api/admin/deployments/"+jobs[0].ID+"/cancel", nil, 204)
	otherAgent.request("POST", "/api/agent/jobs/"+jobs[0].ID+"/status", map[string]string{"status": "downloading"}, 409)
	jobs = decode[[]Deployment](t, admin.request("POST", "/api/admin/deployments", body(other.ID), 201))
	otherAgent.request("POST", "/api/agent/jobs/"+jobs[0].ID+"/status", map[string]string{"status": "failed", "message": "Checksum mismatch"}, 204)
	admin.request("DELETE", "/api/admin/devices/"+other.ID, nil, 204)
}

func TestStatePersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, adminKey)
	if err != nil {
		t.Fatal(err)
	}
	admin := client{t: t, app: s.App(), token: adminKey}
	d, agent := admin.enroll("Persistent", "esp32")
	fw := decode[Firmware](t, admin.request("POST", "/api/admin/firmware", artifact(t, "1", "esp32", "persistent artifact"), 201))
	jobs := decode[[]Deployment](t, admin.request("POST", "/api/admin/deployments", map[string]any{"firmwareId": fw.ID, "deviceIds": []string{d.ID}}, 201))
	if err := s.view(func(st *State) error {
		if st.TokenHashes[d.ID] != hash(agent.token) {
			return fmt.Errorf("Device token not hashed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir, adminKey)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	admin.app = s.App()
	agent.app = admin.app
	state := admin.request("GET", "/api/admin/state", nil, 200)
	if !strings.Contains(string(state), d.ID) || !strings.Contains(string(state), jobs[0].ID) {
		t.Fatal("State was not persisted")
	}
	agent.request("GET", "/api/agent/jobs/"+jobs[0].ID+"/artifact", nil, 200)
}
