package utils

import (
	"sync"
)

type WorkerConfig struct {
	Name               string `json:"name"`
	MaxConcurrentTasks int    `json:"maxConcurrentTasks"`
	ManagerIP          string `json:"managerIP"`
	ManagerPort        string `json:"managerPort"`
	ManagerOauthToken  string `json:"managerOauthToken"`
	OAuthToken         string `json:"oauthToken"`
	Port               string `json:"port"`
	// TaskList           map[string]*globalstructs.Task `json:"taskList"`
	// TaskListMu         sync.Mutex                     `json:"taskListMu"`
	// WorkMutex          sync.Mutex                     `json:"workMutex"`
	// Goroutine          *sync.WaitGroup                `json:"goroutine"`
	// SemaphoreCh        chan struct{}                  `json:"semaphoreCh"`
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

/*
type MessageO struct {
	ID          string   `json:"id"`
	Module      string   `json:"module"`
	Arguments   []string `json:"arguments"`
	CallbackURL string   `json:"callbackURL"`
}
*/
