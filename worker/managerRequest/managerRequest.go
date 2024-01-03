package managerRequest

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	globalstructs "github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/worker/utils"
)

func CreateWebsocket(config *utils.WorkerConfig, verbose, debug bool) (*websocket.Conn, error) {
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
		log.Println("serverAddr", serverAddr)
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true} // InsecureSkipVerify is used for testing purposes only

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
		IddleThreads: config.IddleThreads,
		UP:           true,
	}

	// Marshal the worker object into JSON
	payload, _ := json.Marshal(worker)

	msg := globalstructs.WebsocketMessage{
		Type: "addWorker",
		Json: string(payload),
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
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
		IddleThreads: config.IddleThreads,
		UP:           true,
	}

	// Marshal the worker object into JSON
	payload, _ := json.Marshal(worker)

	msg := globalstructs.WebsocketMessage{
		Type: "deleteWorker",
		Json: string(payload),
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return err
	}

	err = SendMessage(config.Conn, jsonData, verbose, debug, writeLock)
	if err != nil {
		return err
	}

	// Read Response
	_, p, err := config.Conn.ReadMessage()
	if err != nil {
		log.Println(err)
		return err
	}

	err = json.Unmarshal(p, &msg)
	if err != nil {
		log.Println("Error decoding JSON:", err)
		return err
	}

	if msg.Type == "OK" {
		log.Println("Response AddWorker OK from manager")
	} else {
		log.Println("Response AddWorker not OK from manager", msg.Type)
	}

	return nil
}

// CallbackTaskMessage sends a POST request to the manager to callback with a task message
func CallbackTaskMessage(config *utils.WorkerConfig, task *globalstructs.Task, verbose, debug bool, writeLock *sync.Mutex) error {
	// Marshal the task object into JSON
	payload, _ := json.Marshal(task)

	msg := globalstructs.WebsocketMessage{
		Type: "callbackTask",
		Json: string(payload),
	}

	if debug {
		log.Println("msg callback:", msg)
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return err
	}

	err = SendMessage(config.Conn, jsonData, verbose, debug, writeLock)
	if err != nil {
		return err
	}

	return nil
}
