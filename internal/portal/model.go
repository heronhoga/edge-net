package portal

import "time"

type Metrics struct {
	CPU         *float64 `json:"cpu,omitempty"`
	Memory      *float64 `json:"memory,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	Uptime      uint64   `json:"uptime"`
}

type Sample struct {
	At time.Time `json:"at"`
	Metrics
}

type Device struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Kind      string     `json:"kind"`
	Target    string     `json:"target"`
	Location  string     `json:"location"`
	Version   string     `json:"version"`
	CreatedAt time.Time  `json:"createdAt"`
	LastSeen  *time.Time `json:"lastSeen"`
	Metrics   Metrics    `json:"metrics"`
	History   []Sample   `json:"history"`
	Status    string     `json:"status"`
}

type Firmware struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Target    string    `json:"target"`
	SHA256    string    `json:"sha256"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
}

type Deployment struct {
	ID         string    `json:"id"`
	DeviceID   string    `json:"deviceId"`
	DeviceName string    `json:"deviceName"`
	FirmwareID string    `json:"firmwareId"`
	Version    string    `json:"version"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Event struct {
	ID      string    `json:"id"`
	Message string    `json:"message"`
	At      time.Time `json:"at"`
}

type State struct {
	Devices     []Device          `json:"devices"`
	Firmware    []Firmware        `json:"firmware"`
	Deployments []Deployment      `json:"deployments"`
	Events      []Event           `json:"events"`
	TokenHashes map[string]string `json:"tokenHashes"`
}
