package portal

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"
	bolt "go.etcd.io/bbolt"
)

const localWorkspace = "local-admin"
const sessionLifetime = 7 * 24 * time.Hour

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Provider string `json:"provider"`
}

type browserSession struct {
	UserID    string    `json:"userId"`
	CSRFToken string    `json:"csrfToken"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type oauthAttempt struct {
	BrowserHash string    `json:"browserHash"`
	Nonce       string    `json:"nonce"`
	Verifier    string    `json:"verifier"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

func putJSON(tx *bolt.Tx, bucket, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	b, err := tx.CreateBucketIfNotExists([]byte(bucket))
	if err != nil {
		return err
	}
	return b.Put([]byte(key), raw)
}

func getJSON(tx *bolt.Tx, bucket, key string, value any) error {
	b := tx.Bucket([]byte(bucket))
	if b == nil || b.Get([]byte(key)) == nil {
		return fiber.ErrUnauthorized
	}
	return json.Unmarshal(b.Get([]byte(key)), value)
}

func deleteKey(tx *bolt.Tx, bucket, key string) error {
	if b := tx.Bucket([]byte(bucket)); b != nil {
		return b.Delete([]byte(key))
	}
	return nil
}

func pruneExpired(tx *bolt.Tx, bucket string) error {
	b := tx.Bucket([]byte(bucket))
	if b == nil {
		return nil
	}
	cursor := b.Cursor()
	for key, raw := cursor.First(); key != nil; key, raw = cursor.Next() {
		var expiry struct {
			ExpiresAt time.Time `json:"expiresAt"`
		}
		if err := json.Unmarshal(raw, &expiry); err != nil {
			return err
		}
		if !time.Now().Before(expiry.ExpiresAt) {
			if err := cursor.Delete(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Server) createSession(identity googleIdentity, previous string) (string, error) {
	raw := randomID()
	err := s.db.Update(func(tx *bolt.Tx) error {
		if err := pruneExpired(tx, "sessions"); err != nil {
			return err
		}
		// A Google subject, not an email address, is the durable account identifier.
		var id string
		if subjects := tx.Bucket([]byte("googleSubjects")); subjects != nil {
			id = string(subjects.Get([]byte(identity.Subject)))
		}
		if id == "" {
			id = randomID()
			subjects, err := tx.CreateBucketIfNotExists([]byte("googleSubjects"))
			if err != nil {
				return err
			}
			if err := subjects.Put([]byte(identity.Subject), []byte(id)); err != nil {
				return err
			}
		}
		user := User{ID: id, Name: identity.Name, Email: identity.Email, Provider: "google"}
		if err := putJSON(tx, "users", id, user); err != nil {
			return err
		}
		if previous != "" {
			if err := deleteKey(tx, "sessions", hash(previous)); err != nil {
				return err
			}
		}
		return putJSON(tx, "sessions", hash(raw), browserSession{
			UserID: id, CSRFToken: randomID(), ExpiresAt: time.Now().UTC().Add(sessionLifetime),
		})
	})
	return raw, err
}

func (s *Server) readSession(raw string) (User, browserSession, error) {
	var user User
	var session browserSession
	if raw == "" {
		return user, session, fiber.ErrUnauthorized
	}
	err := s.db.View(func(tx *bolt.Tx) error {
		if err := getJSON(tx, "sessions", hash(raw), &session); err != nil {
			return err
		}
		if !time.Now().Before(session.ExpiresAt) {
			return fiber.ErrUnauthorized
		}
		return getJSON(tx, "users", session.UserID, &user)
	})
	return user, session, err
}
