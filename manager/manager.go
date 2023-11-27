// master.go
package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type ManagerConfig struct {
	CallbackURL string            `json:"callbackURL"`
	SlaveURLs   map[string]string `json:"slaveURLs"`
	OAuthToken  string            `json:"oauthToken"`
	Port        string            `json:"port"`
}

type Message struct {
	ID          string   `json:"id"`
	Module      string   `json:"module"`
	Arguments   []string `json:"arguments"`
	CallbackURL string   `json:"callbackURL"`
}

var (
	callbackURL = "http://127.0.0.1:8180/callback/" // Update the callback URL accordingly
	slaveURLs   = map[string]string{
		"slave1": "http://127.0.0.1:8182",
		"slave2": "http://127.0.0.1:8182",
	}
	oauthToken = "your_oauth_tokens" // Replace with your actual OAuth token
	mu         sync.Mutex
	port       = "8080"
)

func loadManagerConfig(filename string) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading manager config file: %s\n", err)
		return
	}

	var config ManagerConfig
	err = json.Unmarshal(content, &config)
	if err != nil {
		fmt.Printf("Error unmarshalling manager config: %s\n", err)
		return
	}

	callbackURL = config.CallbackURL
	slaveURLs = config.SlaveURLs
	oauthToken = config.OAuthToken
	port = config.Port
}

func broadcastMessage(message Message) (string, error) {
	mu.Lock()
	defer mu.Unlock()

	stringResponse := ""
	var err error
	for _, slaveURL := range slaveURLs {
		stringResponse, err = sendToSlave(slaveURL, message)
		if err != nil {
			return "", err
		}
	}
	return stringResponse, nil
}

func sendToSlave(slaveURL string, message Message) (string, error) {
	// Include the callback URL in the message
	message.CallbackURL = callbackURL + message.ID

	payload, _ := json.Marshal(message)

	req, err := http.NewRequest("POST", slaveURL+"/receive", bytes.NewBuffer(payload))
	if err != nil {
		fmt.Printf("Failed to create request to %s: %v\n", slaveURL, err)
		return "", err
	}

	req.Header.Set("Authorization", oauthToken)

	bodyString := ""

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to send message to %s: %v\n", slaveURL, err)
	} else {
		defer response.Body.Close()

		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("Failed to read response body: %v\n", err)
		} else {
			bodyString = string(bodyBytes)
			fmt.Printf("Response Body: %s\n", bodyString)
		}
	}

	return bodyString, nil
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	oauthKey := r.Header.Get("Authorization")
	if oauthKey != oauthToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	recipient := vars["recipient"]
	var message Message
	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate a unique ID for the message
	message.ID = uuid.New().String()

	// Callback URL is now generated dynamically
	message.CallbackURL = callbackURL + message.ID

	response := ""
	if recipient != "" {
		slaveURL, ok := slaveURLs[recipient]
		if !ok {
			http.Error(w, "Invalid recipient", http.StatusBadRequest)
			return
		}
		response, err = sendToSlave(slaveURL, message)
		if err != nil {
			return
		}
	} else {
		response, err = broadcastMessage(message)
		if err != nil {
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, response)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	var result Message
	err := json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		http.Error(w, "Invalid callback body", http.StatusBadRequest)
		return
	}

	fmt.Printf("Received result (ID: %s) from slave:\n%s\n", result.ID, result.Module)

	// Handle the result as needed

	w.WriteHeader(http.StatusOK)
}

func StartManager() {
	fmt.Println("Running as master...")

	loadManagerConfig("manager.conf")

	r := mux.NewRouter()
	r.HandleFunc("/send/{recipient}", handleSendMessage).Methods("POST")
	//r.HandleFunc("/send", handleSendMessage).Methods("POST")
	r.HandleFunc("/callback/{id}", handleCallback).Methods("POST") // Callback endpoint

	http.Handle("/", r)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Println(err)
	}
}
