package utils

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// ManagerConfig manager config file struct
type ManagerConfig struct {
	Users              map[string]string          `json:"users"`
	Workers            map[string]string          `json:"workers"`
	HTTPPort           int                        `json:"httpPort"`
	HTTPSPort          int                        `json:"httpsPort"`
	APIReadTimeout     int                        `json:"apiReadTimeout"`
	APIWriteTimeout    int                        `json:"apiWriteTimeout"`
	APIIdleTimeout     int                        `json:"apiIdleTimeout"`
	DBUsername         string                     `json:"dbUsername"`
	DBPassword         string                     `json:"dbPassword"`
	DBHost             string                     `json:"dbHost"`
	DBPort             string                     `json:"dbPort"`
	DBDatabase         string                     `json:"dbDatabase"`
	StatusCheckSeconds int                        `json:"statusCheckSeconds"`
	StatusCheckDown    int                        `json:"statusCheckDown"`
	DiskPath           string                     `json:"diskPath"`
	CertFolder         string                     `json:"certFolder"`
	ClientHTTP         *http.Client               `json:"clientHTTP"`
	WebSockets         map[string]*websocket.Conn `json:"webSockets"`
}

// ManagerSSHConfig manager SSH config struct
type ManagerSSHConfig struct {
	IPPort             map[string]string `json:"ipPort"`
	SSHUsername        string            `json:"sshUsername"`
	PrivateKeyPath     string            `json:"privateKeyPath"`
	PrivateKeyPassword string            `json:"privateKeyPassword"`
}

// ManagerCloudConfig manager cloud config struct
// https://slugs.do-api.dev/
type ManagerCloudConfig struct {
	Provider     string `json:"provider"`
	APIKey       string `json:"apiKey"`
	SnapshotName string `json:"snapshotName"`
	Servers      int    `json:"servers"`
	Region       string `json:"region"`
	Size         string `json:"size"`
	SSHKeys      string `json:"sshKeys"`
	SSHPort      int    `json:"sshPort"`
	Recreate     bool   `json:"recreate"`
}

// Status General status struct
type Status struct {
	Task   StatusTask   `json:"task"`
	Worker StatusWorker `json:"worker"`
}

// StatusTask task status struct
type StatusTask struct {
	Pending int `json:"pending"`
	Running int `json:"running"`
	Done    int `json:"done"`
	Failed  int `json:"failed"`
	Deleted int `json:"deleted"`
}

// StatusWorker worker status struct
type StatusWorker struct {
	Up   int `json:"up"`
	Down int `json:"down"`
}

// API username middleware
type contextKey string

// UsernameKey key to get username in API
const UsernameKey contextKey = "username"

// WorkerKey key to get worker in API
const WorkerKey contextKey = "worker"
