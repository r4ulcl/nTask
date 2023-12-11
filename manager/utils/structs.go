package utils

type ManagerConfig struct {
	Users         map[string]string `json:"users"`
	Workers       map[string]string `json:"workers"`
	Port          string            `json:"port"`
	DBUsername    string            `json:"dbUsername"`
	DBPassword    string            `json:"dbPassword"`
	DBHost        string            `json:"dbHost"`
	DBPort        string            `json:"dbPort"`
	DBDatabase    string            `json:"dbDatabase"`
	CallbackURL   string            `json:"callbackURL"`
	CallbackToken string            `json:"callbackToken"`
	DiskPath      string            `json:"diskPath"`
}

/*
type MessageOLD struct {
	ID          string   `json:"id"`
	Module      string   `json:"module"`
	Arguments   []string `json:"arguments"`
	CallbackURL string   `json:"callbackURL"`
}
*/
