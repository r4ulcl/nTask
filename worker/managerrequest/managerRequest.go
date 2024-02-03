package managerrequest

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	globalstructs "github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/worker/utils"
)

func CreateWebsocket(config *utils.WorkerConfig, caCertPath string,
	verifyAltName, verbose, debug bool) (*websocket.Conn, error) {

	headers := make(http.Header)
	headers.Set("Authorization", config.ManagerOauthToken)

	var serverAddr string
	if transport, ok := config.ClientHTTP.Transport.(*http.Transport); ok {
		if transport.TLSClientConfig != nil {
			serverAddr = "wss://" + config.ManagerIP + ":" + config.ManagerPort + "/worker/websocket"
		} else {
			serverAddr = "ws://" + config.ManagerIP + ":" + config.ManagerPort + "/worker/websocket"
		}
	} else {
		serverAddr = "wss://" + config.ManagerIP + ":" + config.ManagerPort + "/worker/websocket"
	}

	if debug {
		log.Println("ManagerRequest serverAddr", serverAddr)
	}

	//tlsConfig := &tls.Config{InsecureSkipVerify: false} // InsecureSkipVerify is used for testing purposes only

	tlsConfig, err := utils.GenerateTLSConfig(caCertPath, verifyAltName, verbose, debug)
	if err != nil {
		if debug {
			log.Println("ManagerRequest Error reading worker config file: ", err)
		}
		return nil, err
	}

	dialer := websocket.Dialer{
		TLSClientConfig: tlsConfig,
		Subprotocols:    []string{"chat"},
	}

	conn, _, err := dialer.Dial(serverAddr, headers)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func SendMessage(conn *websocket.Conn, message []byte, verbose, debug bool, writeLock *sync.Mutex) error {
	writeLock.Lock()
	defer writeLock.Unlock()
	if debug {
		log.Println("sendMessage", string(message))
	}
	err := conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		return err
	}
	return nil
}

// AddWorker sends a POST request to add a worker to the manager
func AddWorker(config *utils.WorkerConfig, verbose, debug bool, writeLock *sync.Mutex) error {
	// Create a Worker object with the provided configuration
	worker := globalstructs.Worker{
		Name:         config.Name,
		IddleThreads: config.DefaultThreads,
		UP:           true,
		DownCount:    0,
	}

	// Marshal the worker object into JSON
	payload, _ := json.Marshal(worker)

	msg := globalstructs.WebsocketMessage{
		Type: "addWorker",
		JSON: string(payload),
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Println("Error encoding JSON:", err)
		return err
	}

	err = SendMessage(config.Conn, jsonData, verbose, debug, writeLock)
	if err != nil {
		return err
	}

	return nil
}

// AddWorker sends a POST request to add a worker to the manager
func DeleteWorker(config *utils.WorkerConfig, verbose, debug bool, writeLock *sync.Mutex) error {
	// Create a Worker object with the provided configuration
	worker := globalstructs.Worker{
		Name:         config.Name,
		IddleThreads: -1,
		UP:           true,
		DownCount:    0,
	}

	// Marshal the worker object into JSON
	payload, _ := json.Marshal(worker)

	msg := globalstructs.WebsocketMessage{
		Type: "deleteWorker",
		JSON: string(payload),
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Println("Error encoding JSON:", err)
		return err
	}

	err = SendMessage(config.Conn, jsonData, verbose, debug, writeLock)
	if err != nil {
		return err
	}

	return nil
}

// CallbackTaskMessage sends a POST request to the manager to callback with a task message
func CallbackTaskMessage(config *utils.WorkerConfig, task *globalstructs.Task, verbose, debug bool, writeLock *sync.Mutex) error {
	// Marshal the task object into JSON
	payload, _ := json.Marshal(task)

	msg := globalstructs.WebsocketMessage{
		Type: "callbackTask",
		JSON: string(payload),
	}

	if debug {
		log.Println("ManagerRequest msg callback:", msg)
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Println("Error encoding JSON:", err)
		return err
	}

	err = SendMessage(config.Conn, jsonData, verbose, debug, writeLock)
	if err != nil {
		return err
	}

	return nil
}
