package utils

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type WorkerConfig struct {
	Name              string            `json:"name"`
	IddleThreads      int               `json:"iddleThreads"`
	ManagerIP         string            `json:"managerIP"`
	ManagerPort       string            `json:"managerPort"`
	ManagerOauthToken string            `json:"managerOauthToken"`
	OAuthToken        string            `json:"oauthToken"`
	Port              string            `json:"port"`
	CertFolder        string            `json:"certFolder"`
	InsecureModules   bool              `json:"insecureModules"`
	Modules           map[string]string `json:"modules"`
	ClientHTTP        *http.Client      `json:"clientHTTP"`
	Conn              *websocket.Conn   `json:"Conn"`
}

type Task struct {
	ID          string
	Module      string
	Arguments   []string
	CallbackURL string
	Status      string
	Result      string
	Goroutine   *sync.WaitGroup
}
