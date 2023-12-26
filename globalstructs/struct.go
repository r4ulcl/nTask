package globalstructs

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
	Priority      bool      `json:"priority"`
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
	Priority bool             `json:"priority"`
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
	Name         string `json:"name"`
	IP           string `json:"ip"`
	Port         string `json:"port"`
	OauthToken   string `json:"oauthToken"`
	IddleThreads int    `json:"IddleThreads"`
	UP           bool   `json:"up"`
	DownCount    int    `json:"downCount"`
}

// WorkerStatus struct to process the worker status response.
type WorkerStatus struct {
	IddleThreads int            `json:"IddleThreads"`
	WorkingIDs   map[string]int `json:"workingIds"`
}

type Error struct {
	Error string `json:"error"`
}
