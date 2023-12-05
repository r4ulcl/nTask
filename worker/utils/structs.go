package utils

import (
	"sync"
)

type WorkerConfig struct {
	Name              string            `json:"name"`
	IddleThreads      int               `json:"IddleThreads"`
	ManagerIP         string            `json:"managerIP"`
	ManagerPort       string            `json:"managerPort"`
	ManagerOauthToken string            `json:"managerOauthToken"`
	OAuthToken        string            `json:"oauthToken"`
	Port              string            `json:"port"`
	Modules           map[string]string `json:"modules"`
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
