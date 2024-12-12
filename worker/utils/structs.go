package utils

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// WorkerConfig Worker Config file struct
type WorkerConfig struct {
	Name              string            `json:"name"`
	DefaultThreads    int               `json:"defaultThreads"`
	ManagerIP         string            `json:"managerIP"`
	ManagerPort       int               `json:"managerPort"`
	ManagerOauthToken string            `json:"managerOauthToken"`
	CA                string            `json:"ca"`
	InsecureModules   bool              `json:"insecureModules"`
	Modules           map[string]string `json:"modules"`
	ClientHTTP        *http.Client      `json:"clientHTTP"`
	Conn              *websocket.Conn   `json:"Conn"`
}

// Task Task struct
type Task struct {
	ID          string
	Module      string
	Arguments   []string
	CallbackURL string
	Status      string
	Result      string
	Goroutine   *sync.WaitGroup
}
