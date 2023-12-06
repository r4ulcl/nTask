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
	CallbackURL       string `json:"callbackURL"`
	CallbackToken     string `json:"callbackToken"`
	DiskPath          string `json:"diskPath"`
}

/*
type MessageOLD struct {
	ID          string   `json:"id"`
	Module      string   `json:"module"`
	Arguments   []string `json:"arguments"`
	CallbackURL string   `json:"callbackURL"`
}
*/
