package managerrequest

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	globalstructs "github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/worker/utils"
)

// CreateWebsocket func to create the websocketrs
func CreateWebsocket(config *utils.WorkerConfig, caCertPath string,
	verifyAltName, verbose, debug bool) (*websocket.Conn, error) {

	headers := make(http.Header)
	headers.Set("Authorization", config.ManagerOauthToken)

	var serverAddr string
	portStr := strconv.Itoa(config.ManagerPort)
	if transport, ok := config.ClientHTTP.Transport.(*http.Transport); ok {
		if transport.TLSClientConfig != nil {
			serverAddr = "wss://" + config.ManagerIP + ":" + portStr + "/worker/websocket"
		} else {
			serverAddr = "ws://" + config.ManagerIP + ":" + portStr + "/worker/websocket"
		}
	} else {
		serverAddr = "wss://" + config.ManagerIP + ":" + portStr + "/worker/websocket"
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

// SendMessage funct to send message to a websocket from a worker
func SendMessage(conn *websocket.Conn, message []byte, verbose, debug bool, writeLock *sync.Mutex) error {
	writeLock.Lock()
	defer writeLock.Unlock()
	if debug {
		log.Println("sendMessage:", string(message))
	}
	err := conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		return err
	}
	return nil
}

// SendWebSocketMessage is a helper function to send a WebSocket message to the manager
func SendWebSocketMessage(config *utils.WorkerConfig, messageType string, payload interface{}, verbose, debug bool, writeLock *sync.Mutex) error {
	// Marshal the payload into JSON
	payloadData, err := json.Marshal(payload)
	if err != nil {
		log.Println("Error encoding JSON payload:", err)
		return err
	}

	// Create the WebSocket message
	msg := globalstructs.WebsocketMessage{
		Type: messageType,
		JSON: string(payloadData),
	}

	// Debug logging
	if debug {
		log.Printf("ManagerRequest msg (%s): %v", messageType, msg)
	}

	// Marshal the WebSocket message into JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Println("Error encoding WebSocket message:", err)
		return err
	}

	// Send the message
	return SendMessage(config.Conn, jsonData, verbose, debug, writeLock)
}

// AddWorker sends a POST request to add a worker to the manager
func AddWorker(config *utils.WorkerConfig, verbose, debug bool, writeLock *sync.Mutex) error {
	worker := globalstructs.Worker{
		Name:           config.Name,
		DefaultThreads: config.DefaultThreads,
		IddleThreads:   config.DefaultThreads,
		UP:             true,
		DownCount:      0,
	}

	return SendWebSocketMessage(config, "addWorker", worker, verbose, debug, writeLock)
}

// DeleteWorker sends a POST request to delete a worker from the manager
func DeleteWorker(config *utils.WorkerConfig, verbose, debug bool, writeLock *sync.Mutex) error {
	worker := globalstructs.Worker{
		Name:         config.Name,
		IddleThreads: -1,
		UP:           true,
		DownCount:    0,
	}

	return SendWebSocketMessage(config, "deleteWorker", worker, verbose, debug, writeLock)
}

// CallbackTaskMessage sends a POST request to the manager with a task message
func CallbackTaskMessage(config *utils.WorkerConfig, task *globalstructs.Task, verbose, debug bool, writeLock *sync.Mutex) error {
	return SendWebSocketMessage(config, "callbackTask", task, verbose, debug, writeLock)
}
