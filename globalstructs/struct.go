package globalstructs

import "github.com/gorilla/websocket"

// package for structs used in manager and workers
// in case I want to separate the project one day

// Task Struct to store all Task information.
type Task struct {
	ID            string    `json:"id"`
	Commands      []Command `json:"command"`
	Name          string    `json:"name"`
	CreatedAt     string    `json:"createdAt"`
	UpdatedAt     string    `json:"updatedAt"`
	ExecutedAt    string    `json:"executedAt"`
	Status        string    `json:"status"` // pending, running, done, failed, deleted
	WorkerName    string    `json:"workerName"`
	Username      string    `json:"username"`
	Priority      int       `json:"priority"`
	CallbackURL   string    `json:"callbackURL"`
	CallbackToken string    `json:"callbackToken"`
}

// Command struct for Commands in a task
type Command struct {
	Module         string `json:"module"`
	Args           string `json:"args"`
	FileContent    string `json:"fileContent"`
	RemoteFilePath string `json:"remoteFilePath"`
	Output         string `json:"output"`
}

// Task Struct for swagger docs, for the POST
type TaskSwagger struct {
	Commands []CommandSwagger `json:"command"`
	Name     string           `json:"name"`
	Priority int              `json:"priority"`
}

// Command struct for swagger documentation
type CommandSwagger struct {
	Module         string `json:"module"`
	Args           string `json:"args"`
	FileContent    string `json:"fileContent"`
	RemoteFilePath string `json:"remoteFilePath"`
}

// Worker struct to store all worker information.
type Worker struct {
	// Workers name (unique)
	Name           string `json:"name"`
	DefaultThreads int    `json:"defaultThreads"`
	IddleThreads   int    `json:"iddleThreads"`
	UP             bool   `json:"up"`
	DownCount      int    `json:"downCount"`
}

// WorkerStatus struct to process the worker status response.
type WorkerStatus struct {
	Name         string         `json:"name"`
	IddleThreads int            `json:"iddleThreads"`
	WorkingIDs   map[string]int `json:"workingIds"`
}

type Error struct {
	Error string `json:"error"`
}

// websockets

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  4096, // 4 kilobytes
	WriteBufferSize: 4096, // 4 kilobytes
}

type WebsocketMessage struct {
	Type string `json:"type"`
	JSON string `json:"json"`
}
