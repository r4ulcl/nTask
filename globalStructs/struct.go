package globalStructs

// package for structs used in manager and workers
// in case I want to separate the project one day

type Task struct {
	ID         string   `json:"id"`
	Module     string   `json:"module"`
	Args       []string `json:"args"`
	Created_at string   `json:"created_at"`
	Updated_at string   `json:"updated_at"`
	Status     string   `json:"status"` //pending, running, done, failed, deleted
	WorkerName string   `json:"workerName"`
	Output     string   `json:"output"`
	Priority   bool     `json:"priority"`
}

// swagger:parameters myEndpoint
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

type WorkerStatus struct {
	Working   bool   `json:"working"`
	WorkingID string `json:"workingID"`
}
