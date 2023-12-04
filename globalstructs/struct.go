package globalstructs

// package for structs used in manager and workers
// in case I want to separate the project one day

// Task Struct to store all Task information
type Task struct {
	ID         string   `json:"id"`
	Module     string   `json:"module"`
	Args       []string `json:"args"`
	CreatedAt  string   `json:"createdAt"`
	UpdatedAt  string   `json:"updatedAt"`
	Status     string   `json:"status"` // pending, running, done, failed, deleted
	WorkerName string   `json:"workerName"`
	Output     string   `json:"output"`
	Priority   bool     `json:"priority"`
}

// Worker struct to store all worker informacion
type Worker struct {
	// Workers name (unique)
	Name       string `json:"name"`
	IP         string `json:"ip"`
	Port       string `json:"port"`
	OauthToken string `json:"oauthToken"`
	Working    bool   `json:"working"`
	UP         bool   `json:"up"`
	Count      int    `json:"count"`
}

// WorkerStatus struct to process the worker status response
type WorkerStatus struct {
	Working   bool   `json:"working"`
	WorkingID string `json:"workingID"`
}
