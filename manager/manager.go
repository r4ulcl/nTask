// manager.go
package manager

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/gorilla/mux"
	"github.com/r4ulcl/NetTask/manager/database"
	"github.com/r4ulcl/NetTask/manager/utils"
)

func loadManagerConfig(filename string) (utils.ManagerConfig, error) {
	var config utils.ManagerConfig

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading manager config file: %s\n", err)
		return config, err
	}

	err = json.Unmarshal(content, &config)
	if err != nil {
		fmt.Printf("Error unmarshalling manager config: %s\n", err)
		return config, err
	}

	return config, nil
}

func incorrectOauth(clientOauthKey, oauthToken string) bool {
	return clientOauthKey != oauthToken
}

func incorrectOauthWorker(clientOauthKey, oauthTokenWorkers string) bool {
	return clientOauthKey != oauthTokenWorkers
}

//verifyWorkersLoop check and set if the workers are UP infinite
func verifyWorkersLoop(OauthTokenWorkers string, db *sql.DB) {
	for {
		verifyWorkers(OauthTokenWorkers, db)
		time.Sleep(5 * time.Second)
	}
}

//verifyWorkers check and set if the workers are UP
func verifyWorkers(OauthTokenWorkers string, db *sql.DB) {

	workers, err := database.GetWorkers(db)
	if err != nil {
		fmt.Println(err)
	}
	for _, worker := range workers {
		verifyWorker(db, OauthTokenWorkers, worker)
	}

}

//verifyWorker check and set if the workers are UP
func verifyWorker(db *sql.DB, OauthTokenWorkers string, worker utils.Worker) error {
	workerURL := "http://" + worker.IP + ":" + worker.Port

	// Create an HTTP client and send a GET request
	client := &http.Client{}
	req, err := http.NewRequest("GET", workerURL+"/status", nil)
	if err != nil {
		fmt.Printf("Failed to create request to %s: %v\n", workerURL, err)
		return err
	}

	req.Header.Set("Authorization", OauthTokenWorkers)

	resp, err := client.Do(req)
	if err != nil {
		//fmt.Println("Error making request:", err)
		//if error making request is offline!
		database.SetWorkerUPto(false, db, worker)
		return err
	}
	defer resp.Body.Close()

	//if no error making request is online!
	database.SetWorkerUPto(true, db, worker)

	// Read the response body into a byte slice
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return err
	}

	// Unmarshal the JSON into a TaskResponse struct
	var status utils.WorkerStatusResponse
	err = json.Unmarshal(body, &status)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return err
	}

	if status.Working != worker.Working {
		fmt.Println("DIFFERENT!")
	} else {
		fmt.Println("SAME!")

	}

	return nil

}

/*func sendToSlave(slaveURL string, message Message) (string, error) {
	// Include the callback URL in the message
	message.CallbackURL = callbackURL + message.ID

	payload, _ := json.Marshal(message)

	req, err := http.NewRequest("POST", slaveURL+"/task", bytes.NewBuffer(payload))
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
}*/

/*
// @Summary Send a message
// @Description Send a message to a recipient or broadcast to all recipients
// @Tags messages
// @Accept json
// @Produce json
// @Param Authorization header string true "OAuth Key" default(WLJ2xVQZ5TXVw4qEznZDnmEEV)
// @Param recipient path string false "Recipient ID"
// @Success 200 {string} string "OK"
// @Failure 400 {string} string "Bad Request"
// @Failure 401 {string} string "Unauthorized"
// @Router /messages/send/{recipient} [post]
func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	oauthKey := r.Header.Get("Authorization")
	if incorrectOauth(oauthKey) {
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
}*/

func StartManager() {
	fmt.Println("Running as manager...")

	config, err := loadManagerConfig("manager.conf")
	if err != nil {
		fmt.Println(err)
	}

	// Start DB
	db, err := database.ConnectDB(config.DBUsername, config.DBPassword, config.DBHost, config.DBPort, config.DBDatabase)
	if err != nil {
		fmt.Println(err)
	}

	//verify status workers infinite
	go verifyWorkersLoop(config.OauthTokenWorkers, db)

	r := mux.NewRouter()

	// Serve Swagger UI at /swagger
	// Serve Swagger UI at /swagger
	r.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/docs/swagger.json"), // URL to the swagger.json file
	))

	// Serve Swagger JSON at /swagger/doc.json
	r.HandleFunc("/docs/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs/swagger.json")
	}).Methods("GET")

	//r.HandleFunc("/send/{recipient}", handleSendMessage).Methods("POST")

	// CallBack
	//r.HandleFunc("/callback/{id}", handleCallback).Methods("POST") // Callback endpoint

	// worker
	r.HandleFunc("/worker", func(w http.ResponseWriter, r *http.Request) {
		handleWorkerGet(w, r, config, db)
	}).Methods("GET") //get workers

	r.HandleFunc("/worker", func(w http.ResponseWriter, r *http.Request) {
		handleWorkerAdd(w, r, config, db)
	}).Methods("POST") //add worker

	r.HandleFunc("/worker/{NAME}", func(w http.ResponseWriter, r *http.Request) {
		handleWorkerRMName(w, r, config, db)
	}).Methods("DELETE") //delete worker

	r.HandleFunc("/worker/{NAME}", func(w http.ResponseWriter, r *http.Request) {
		handleWorkerStatus(w, r, config, db)
	}).Methods("GET") //check status 1 worker

	/*
		// task
		r.HandleFunc("/task", handleTask).Methods("GET")
		r.HandleFunc("/task/add", handleTaskAdd).Methods("POST")
		r.HandleFunc("/task/stop", handletasktop).Methods("POST")
		r.HandleFunc("/task/rm", handleTaskRM).Methods("POST")
		r.HandleFunc("/task/status/{id}", handletasktatus).Methods("GET")

		// vuln
		r.HandleFunc("/vuln/", handleDummy).Methods("GET")
		r.HandleFunc("/vuln/add", handleDummy).Methods("POST")
		r.HandleFunc("/vuln/rm", handleDummy).Methods("POST")
		r.HandleFunc("/vuln/info/{id}", handleDummy).Methods("GET")

		// Scope

		// Asset
	*/

	http.Handle("/", r)
	err = http.ListenAndServe(":"+config.Port, nil)
	if err != nil {
		fmt.Println(err)
	}
	/*
		err = http.ListenAndServe(":"+config.Port, allowCORS(http.DefaultServeMux))
		if err != nil {
			fmt.Println(err)
		}
	*/

}

// allowCORS is a middleware function that adds CORS headers to the response.
func allowCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization") // Add Authorization header

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Call the next handler in the chain
		handler.ServeHTTP(w, r)
	})
}
