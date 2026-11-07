package portal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Server struct {
	db         *bolt.DB
	dir        string
	adminToken string
}

func Open(dir, token string) (*Server, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(abs, "artifacts"), 0700); err != nil {
		return nil, err
	}
	db, err := bolt.Open(filepath.Join(abs, "edgenet.db"), 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	s := &Server{db: db, dir: abs, adminToken: token}
	err = s.change(func(st *State) error { return nil })
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Server) Close() error { return s.db.Close() }

func load(tx *bolt.Tx) (*State, error) {
	st := &State{Devices: []Device{}, Firmware: []Firmware{}, Deployments: []Deployment{}, Events: []Event{}, TokenHashes: map[string]string{}}
	if b := tx.Bucket([]byte("portal")); b != nil {
		if raw := b.Get([]byte("state")); raw != nil {
			if err := json.Unmarshal(raw, st); err != nil {
				return nil, err
			}
		}
	}
	return st, nil
}

func (s *Server) view(fn func(*State) error) error {
	return s.db.View(func(tx *bolt.Tx) error {
		st, err := load(tx)
		if err != nil {
			return err
		}
		return fn(st)
	})
}

// bbolt serializes writers and commits state atomically, including job transitions.
func (s *Server) change(fn func(*State) error) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		st, err := load(tx)
		if err != nil {
			return err
		}
		if err := fn(st); err != nil {
			return err
		}
		raw, err := json.Marshal(st)
		if err != nil {
			return err
		}
		b, err := tx.CreateBucketIfNotExists([]byte("portal"))
		if err != nil {
			return err
		}
		return b.Put([]byte("state"), raw)
	})
}
