package utils

type ManagerConfig struct {
	OAuthToken        string `json:"oauthToken"`
	OauthTokenWorkers string `json:"oauthTokenWorkers"`
	Port              string `json:"port"`
	DBUsername        string `json:"dbUsername"`
	DBPassword        string `json:"dbPassword"`
	DBHost            string `json:"dbHost"`
	DBPort            string `json:"dbPort"`
	DBDatabase        string `json:"dbDatabase"`
}

type WorkerStatusResponse struct {
	Working   bool   `json:"working"`
	MessageID string `json:"messageID"`
}

type MessageOLD struct {
	ID          string   `json:"id"`
	Module      string   `json:"module"`
	Arguments   []string `json:"arguments"`
	CallbackURL string   `json:"callbackURL"`
}

// swagger:parameters myEndpoint
type Worker struct {
	// Workers name (unique)
	Name    string `json:"name"`
	IP      string `json:"ip"`
	Port    string `json:"port"`
	Working bool   `json:"working"`
	UP      bool   `json:"up"`
}

type Task struct {
	ID         string `json:"id"`
	Created_at string `json:"created_at"`
	Updated_at string `json:"updated_at"`
	Status     string `json:"status"`
	WorkerName string `json:"workerName"`
	Output     string `json:"output"`
}
