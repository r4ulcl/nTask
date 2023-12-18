package utils

import (
	"net/http"
	"sync"
)

type WorkerConfig struct {
	Name              string            `json:"name"`
	IddleThreads      int               `json:"iddleThreads"`
	ManagerIP         string            `json:"managerIP"`
	ManagerPort       string            `json:"managerPort"`
	ManagerOauthToken string            `json:"managerOauthToken"`
	OAuthToken        string            `json:"oauthToken"`
	Port              string            `json:"port"`
	InsecureModules   bool              `json:"insecureModules"`
	Modules           map[string]string `json:"modules"`
	ClientHTTP        *http.Client      `json:"clientHTTP"`
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
