package portal

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
)

const maxArtifact = 64 << 20

func randomID() string {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

func hash(v string) string             { b := sha256.Sum256([]byte(v)); return hex.EncodeToString(b[:]) }
func equal(a, b string) bool           { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }
func bearer(c fiber.Ctx) string        { return strings.TrimPrefix(c.Get("Authorization"), "Bearer ") }
func bad(msg string) error             { return fiber.NewError(400, msg) }
func conflict(msg string) error        { return fiber.NewError(409, msg) }
func validText(v string, max int) bool { return strings.TrimSpace(v) != "" && len(v) <= max }

func addEvent(st *State, message string) {
	st.Events = append([]Event{{ID: randomID(), Message: message, At: time.Now().UTC()}}, st.Events...)
	if len(st.Events) > 100 {
		st.Events = st.Events[:100]
	}
}

func device(st *State, id string) (*Device, error) {
	for i := range st.Devices {
		if st.Devices[i].ID == id {
			return &st.Devices[i], nil
		}
	}
	return nil, fiber.NewError(404, "Device not found")
}

func firmware(st *State, id string) (*Firmware, error) {
	for i := range st.Firmware {
		if st.Firmware[i].ID == id {
			return &st.Firmware[i], nil
		}
	}
	return nil, fiber.NewError(404, "Firmware not found")
}

func active(status string) bool {
	return status == "queued" || status == "downloading" || status == "installing"
}

func (s *Server) App() *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "Edgenet", BodyLimit: maxArtifact + (1 << 20), Immutable: true,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code, message := 500, "Internal server error"
			var fe *fiber.Error
			if errors.As(err, &fe) {
				code, message = fe.Code, fe.Message
			} else {
				log.Printf("request error: %v", err)
			}
			return c.Status(code).JSON(fiber.Map{"error": message})
		},
	})
	app.Use(recover.New())
	app.Use("/api", func(c fiber.Ctx) error { c.Set("Cache-Control", "no-store"); return c.Next() })
	app.Get("/api/health", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	admin := app.Group("/api/admin", func(c fiber.Ctx) error {
		if !equal(hash(bearer(c)), hash(s.adminToken)) {
			return fiber.ErrUnauthorized
		}
		return c.Next()
	})
	admin.Get("/state", s.snapshot)
	admin.Post("/devices", s.enroll)
	admin.Post("/devices/:id/rotate-token", s.rotateToken)
	admin.Delete("/devices/:id", s.removeDevice)
	admin.Post("/firmware", s.upload)
	admin.Post("/deployments", s.deploy)
	admin.Post("/deployments/:id/cancel", s.cancel)
	agent := app.Group("/api/agent", func(c fiber.Ctx) error {
		id := c.Get("X-Device-ID")
		if err := s.view(func(st *State) error {
			stored, ok := st.TokenHashes[id]
			if !ok || !equal(stored, hash(bearer(c))) {
				return fiber.ErrUnauthorized
			}
			return nil
		}); err != nil {
			return err
		}
		c.Locals("deviceID", id)
		return c.Next()
	})
	agent.Post("/heartbeat", s.heartbeat)
	agent.Get("/jobs", s.jobs)
	agent.Post("/jobs/:id/status", s.report)
	agent.Get("/jobs/:id/artifact", s.download)
	app.Use("/api", func(c fiber.Ctx) error { return fiber.ErrNotFound })
	app.Use(static.New("dist"))
	return app
}

func (s *Server) snapshot(c fiber.Ctx) error {
	return s.view(func(st *State) error {
		for i := range st.Devices {
			d := &st.Devices[i]
			d.Status = "pending"
			if d.LastSeen != nil {
				d.Status = "offline"
				if time.Since(*d.LastSeen) < 90*time.Second {
					d.Status = "online"
				}
			}
		}
		return c.JSON(fiber.Map{"devices": st.Devices, "firmware": st.Firmware, "deployments": st.Deployments, "events": st.Events, "serverTime": time.Now().UTC()})
	})
}

func (s *Server) enroll(c fiber.Ctx) error {
	var in struct {
		Name     string `json:"name"`
		Kind     string `json:"kind"`
		Target   string `json:"target"`
		Location string `json:"location"`
	}
	if err := c.Bind().Body(&in); err != nil {
		return bad("Invalid device payload")
	}
	in.Name, in.Target, in.Location = strings.TrimSpace(in.Name), strings.TrimSpace(in.Target), strings.TrimSpace(in.Location)
	if !validText(in.Name, 80) || !validText(in.Target, 100) || len(in.Location) > 100 {
		return bad("Name and hardware target are required (80 and 100 characters maximum)")
	}
	switch in.Kind {
	case "esp32", "arduino":
	default:
		return bad("Unsupported device kind")
	}
	token := randomID()
	d := Device{ID: randomID(), Name: in.Name, Kind: in.Kind, Target: in.Target, Location: in.Location, Version: "unknown", CreatedAt: time.Now().UTC(), History: []Sample{}, Status: "pending"}
	if err := s.change(func(st *State) error {
		st.Devices = append(st.Devices, d)
		st.TokenHashes[d.ID] = hash(token)
		addEvent(st, d.Name+" enrolled")
		return nil
	}); err != nil {
		return err
	}
	return c.Status(201).JSON(fiber.Map{"device": d, "token": token})
}

func (s *Server) rotateToken(c fiber.Ctx) error {
	token := randomID()
	if err := s.change(func(st *State) error {
		d, err := device(st, c.Params("id"))
		if err != nil {
			return err
		}
		st.TokenHashes[d.ID] = hash(token)
		addEvent(st, "Access token rotated for "+d.Name)
		return nil
	}); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"token": token})
}

func (s *Server) removeDevice(c fiber.Ctx) error {
	if err := s.change(func(st *State) error {
		d, err := device(st, c.Params("id"))
		if err != nil {
			return err
		}
		for _, j := range st.Deployments {
			if j.DeviceID == d.ID && active(j.Status) {
				return conflict("Resolve the active deployment before removing this device")
			}
		}
		addEvent(st, d.Name+" removed")
		delete(st.TokenHashes, d.ID)
		for i := range st.Devices {
			if st.Devices[i].ID == d.ID {
				st.Devices = append(st.Devices[:i], st.Devices[i+1:]...)
				break
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return c.SendStatus(204)
}

func (s *Server) heartbeat(c fiber.Ctx) error {
	var in struct {
		Version string  `json:"version"`
		Metrics Metrics `json:"metrics"`
	}
	if err := c.Bind().Body(&in); err != nil {
		return bad("Invalid heartbeat")
	}
	if !validText(in.Version, 80) {
		return bad("Firmware version is required")
	}
	for _, value := range []*float64{in.Metrics.CPU, in.Metrics.Memory} {
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 || *value > 100) {
			return bad("CPU and memory must be percentages from 0 to 100")
		}
	}
	if v := in.Metrics.Temperature; v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0) || *v < -100 || *v > 300) {
		return bad("Invalid temperature")
	}
	if err := s.change(func(st *State) error {
		d, err := device(st, c.Locals("deviceID").(string))
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if d.LastSeen == nil || now.Sub(*d.LastSeen) >= 90*time.Second {
			addEvent(st, d.Name+" connected")
		}
		d.LastSeen, d.Version, d.Metrics = &now, in.Version, in.Metrics
		d.History = append(d.History, Sample{At: now, Metrics: in.Metrics})
		if len(d.History) > 60 {
			d.History = d.History[len(d.History)-60:]
		}
		return nil
	}); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"heartbeatIntervalSeconds": 30})
}

func (s *Server) upload(c fiber.Ctx) error {
	version, target := strings.TrimSpace(c.FormValue("version")), strings.TrimSpace(c.FormValue("target"))
	if !validText(version, 80) || !validText(target, 100) {
		return bad("Version and hardware target are required")
	}
	fh, err := c.FormFile("file")
	if err != nil || fh.Size <= 0 || fh.Size > maxArtifact {
		return bad("Upload a non-empty artifact up to 64 MiB")
	}
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	id := randomID()
	path := filepath.Join(s.dir, "artifacts", id)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(path)
		}
	}()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(src, maxArtifact+1))
	if err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if n == 0 || n > maxArtifact {
		return bad("Invalid artifact size")
	}
	fw := Firmware{ID: id, Name: filepath.Base(fh.Filename), Version: version, Target: target, SHA256: hex.EncodeToString(h.Sum(nil)), Size: n, CreatedAt: time.Now().UTC()}
	if err := s.change(func(st *State) error {
		for _, existing := range st.Firmware {
			if existing.Target == target && existing.Version == version {
				return conflict("This target already has an artifact with that version")
			}
		}
		st.Firmware = append([]Firmware{fw}, st.Firmware...)
		addEvent(st, "Firmware "+version+" uploaded for "+target)
		return nil
	}); err != nil {
		return err
	}
	committed = true
	return c.Status(201).JSON(fw)
}

func (s *Server) deploy(c fiber.Ctx) error {
	var in struct {
		FirmwareID string   `json:"firmwareId"`
		DeviceIDs  []string `json:"deviceIds"`
	}
	if err := c.Bind().Body(&in); err != nil || len(in.DeviceIDs) == 0 || len(in.DeviceIDs) > 100 {
		return bad("Select between 1 and 100 devices")
	}
	jobs := []Deployment{}
	if err := s.change(func(st *State) error {
		fw, err := firmware(st, in.FirmwareID)
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, id := range in.DeviceIDs {
			if seen[id] {
				return bad("Duplicate device ID")
			}
			seen[id] = true
			d, err := device(st, id)
			if err != nil {
				return err
			}
			if d.Target != fw.Target {
				return conflict(d.Name + " has an incompatible hardware target")
			}
			for _, j := range st.Deployments {
				if j.DeviceID == id && active(j.Status) {
					return conflict(d.Name + " already has an active deployment")
				}
			}
			now := time.Now().UTC()
			jobs = append(jobs, Deployment{ID: randomID(), DeviceID: id, DeviceName: d.Name, FirmwareID: fw.ID, Version: fw.Version, Status: "queued", CreatedAt: now, UpdatedAt: now})
		}
		st.Deployments = append(jobs, st.Deployments...)
		addEvent(st, fmt.Sprintf("Firmware %s queued for %d device(s)", fw.Version, len(jobs)))
		return nil
	}); err != nil {
		return err
	}
	return c.Status(201).JSON(jobs)
}

func (s *Server) cancel(c fiber.Ctx) error {
	if err := s.change(func(st *State) error {
		for i := range st.Deployments {
			j := &st.Deployments[i]
			if j.ID != c.Params("id") {
				continue
			}
			if j.Status != "queued" {
				return conflict("Only queued deployments can be cancelled")
			}
			j.Status, j.UpdatedAt = "cancelled", time.Now().UTC()
			addEvent(st, "Deployment cancelled for "+j.DeviceName)
			return nil
		}
		return fiber.ErrNotFound
	}); err != nil {
		return err
	}
	return c.SendStatus(204)
}

func (s *Server) jobs(c fiber.Ctx) error {
	return s.view(func(st *State) error {
		jobs := []fiber.Map{}
		for _, j := range st.Deployments {
			if j.DeviceID != c.Locals("deviceID").(string) || !active(j.Status) {
				continue
			}
			fw, err := firmware(st, j.FirmwareID)
			if err != nil {
				return err
			}
			jobs = append(jobs, fiber.Map{"deployment": j, "firmware": fw, "downloadPath": "/api/agent/jobs/" + j.ID + "/artifact"})
		}
		return c.JSON(jobs)
	})
}

func (s *Server) report(c fiber.Ctx) error {
	var in struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := c.Bind().Body(&in); err != nil || len(in.Message) > 500 {
		return bad("Invalid status report")
	}
	if err := s.change(func(st *State) error {
		for i := range st.Deployments {
			j := &st.Deployments[i]
			if j.ID != c.Params("id") || j.DeviceID != c.Locals("deviceID").(string) {
				continue
			}
			if j.Status == in.Status {
				return nil
			} // Reports can be retried after a lost response.
			allowed := (j.Status == "queued" && in.Status == "downloading") || (j.Status == "downloading" && in.Status == "installing") || (j.Status == "installing" && in.Status == "succeeded") || (active(j.Status) && in.Status == "failed")
			if !allowed {
				return conflict("Invalid deployment status transition")
			}
			j.Status, j.Message, j.UpdatedAt = in.Status, in.Message, time.Now().UTC()
			if in.Status == "succeeded" {
				d, err := device(st, j.DeviceID)
				if err != nil {
					return err
				}
				d.Version = j.Version
			}
			addEvent(st, j.DeviceName+": deployment "+in.Status)
			return nil
		}
		return fiber.ErrNotFound
	}); err != nil {
		return err
	}
	return c.SendStatus(204)
}

func (s *Server) download(c fiber.Ctx) error {
	var path string
	if err := s.view(func(st *State) error {
		for _, j := range st.Deployments {
			if j.ID == c.Params("id") && j.DeviceID == c.Locals("deviceID").(string) && active(j.Status) {
				path = filepath.Join(s.dir, "artifacts", j.FirmwareID)
				return nil
			}
		}
		return fiber.ErrNotFound
	}); err != nil {
		return err
	}
	c.Set("Content-Type", "application/octet-stream")
	return c.SendFile(path)
}
