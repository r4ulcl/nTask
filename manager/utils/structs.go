package utils

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type ManagerConfig struct {
	Users      map[string]string          `json:"users"`
	Workers    map[string]string          `json:"workers"`
	Port       string                     `json:"port"`
	DBUsername string                     `json:"dbUsername"`
	DBPassword string                     `json:"dbPassword"`
	DBHost     string                     `json:"dbHost"`
	DBPort     string                     `json:"dbPort"`
	DBDatabase string                     `json:"dbDatabase"`
	DiskPath   string                     `json:"diskPath"`
	CertFolder string                     `json:"certFolder"`
	ClientHTTP *http.Client               `json:"clientHTTP"`
	WebSockets map[string]*websocket.Conn `json:"webSockets"`
}

/*
type MessageOLD struct {
	ID          string   `json:"id"`
	Module      string   `json:"module"`
	Arguments   []string `json:"arguments"`
	CallbackURL string   `json:"callbackURL"`
}
*/

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
