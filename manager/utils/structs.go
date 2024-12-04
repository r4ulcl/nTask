package utils

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type ManagerConfig struct {
	Users              map[string]string          `json:"users"`
	Workers            map[string]string          `json:"workers"`
	HttpPort           string                     `json:"httpPort"`
	HttpsPort          string                     `json:"httpsPort"`
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

type ManagerSSHConfig struct {
	IPPort             map[string]string `json:"ipPort"`
	SSHUsername        string            `json:"sshUsername"`
	PrivateKeyPath     string            `json:"privateKeyPath"`
	PrivateKeyPassword string            `json:"privateKeyPassword"`
}

// https://slugs.do-api.dev/
type ManagerCloudConfig struct {
	Provider     string `json:"provider"`
	ApiKey       string `json:"apiKey"`
	SnapshotName string `json:"snapshotName"`
	Servers      int    `json:"servers"`
	Region       string `json:"region"`
	Size         string `json:"size"`
	SshKeys      string `json:"sshKeys"`
	SshPort      int    `json:"sshPort"`
	Recreate     bool   `json:"recreate"`
}

type Status struct {
	Task   StatusTask   `json:"task"`
	Worker StatusWorker `json:"worker"`
}

type StatusTask struct {
	Pending int `json:"pending"`
	Running int `json:"running"`
	Done    int `json:"done"`
	Failed  int `json:"failed"`
	Deleted int `json:"deleted"`
}

type StatusWorker struct {
	Up   int `json:"up"`
	Down int `json:"down"`
}

// API username middleware
type contextKey string

const UsernameKey contextKey = "username"
const WorkerKey contextKey = "worker"
