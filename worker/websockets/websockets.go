package websockets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/worker/managerrequest"
	"github.com/r4ulcl/nTask/worker/process"
	"github.com/r4ulcl/nTask/worker/utils"
)

func SetConn(conn *websocket.Conn, config *utils.WorkerConfig, verbose, debug bool, writeLock *sync.Mutex) {
	writeLock.Lock()
	defer writeLock.Unlock()

	config.Conn = conn

}

func GetMessage(config *utils.WorkerConfig, status *globalstructs.WorkerStatus, verbose, debug bool, writeLock *sync.Mutex) {
	for {

		response := globalstructs.WebsocketMessage{
			Type: "",
			JSON: "",
		}

		_, p, err := config.Conn.ReadMessage() //messageType
		if err != nil {
			log.Println("WebSockets config.Conn.ReadMessage()", err)
			time.Sleep(time.Second * 5)

			continue
		}

		var msg globalstructs.WebsocketMessage
		err = json.Unmarshal(p, &msg)
		if err != nil {
			log.Println("WebSockets Error decoding JSON:", err)
			continue
		}

		switch msg.Type {

		case "status":
			if debug {
				log.Println("Status message recieve")
			}
			response, err = messageStatusTask(config, status, msg, verbose, debug)
			if err != nil {
				log.Println("status error: ", err)
			}
		case "addTask":
			response, err = messageAddTask(config, status, msg, verbose, debug, writeLock)
			if err != nil {
				log.Println("addTask error: ", err)
			}
		case "deleteTask":
			response, err = messageDeleteTask(config, status, msg, verbose, debug, writeLock)
			if err != nil {
				log.Println("deleteTask error: ", err)
			}
		}

		if debug {
			log.Printf("Received message type: %s\n", msg.Type)
			log.Printf("Received message json: %s\n", msg.JSON)
		}

		if response.Type != "" {
			jsonData, err := json.Marshal(response)
			if err != nil {
				log.Println("WebSockets Marshal error: ", err)
			}
			err = managerrequest.SendMessage(config.Conn, jsonData, verbose, debug, writeLock)
			if err != nil {
				log.Println("WebSockets SendMessage error: ", err)
			}
		}
	}
}

func RecreateConnection(config *utils.WorkerConfig, verifyAltName, verbose, debug bool, writeLock *sync.Mutex) {
	// Send Ping message every 5 seconds

	// Set Pong handler
	pongReceived := make(chan struct{})

	config.Conn.SetPongHandler(func(appData string) error {
		if debug {
			log.Println("Received Pong:", appData)
		}
		// Notify that Pong has been received
		pongReceived <- struct{}{}
		return nil
	})

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if debug {
				log.Println("Checking RecreateConnection")
			}

			// Use a channel to handle the timeout
			timeout := time.After(5 * time.Second)

			err := config.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
			if err != nil {
				log.Println("Error sending Ping err:", err)
				CreateConnection(config, verifyAltName, verbose, debug, writeLock)
			} else {
				if debug {
					log.Println("RecreateConnection - Connection ok")
				}
			}

			// Wait for Pong or timeout
			select {
			case <-pongReceived:
				if debug {
					log.Println("pongReceived")
				}
				// Pong received, continue the loop
			case <-timeout:
				// Pong not received within the timeout, close the connection
				log.Println("Pong not received within the timeout. Closing the connection.")
				err := config.Conn.Close()
				if err != nil {
					log.Println("Error closing connection:", err)
				}
				CreateConnection(config, verifyAltName, verbose, debug, writeLock)

			}

		}
	}
}

func CreateConnection(config *utils.WorkerConfig, verifyAltName, verbose, debug bool, writeLock *sync.Mutex) {
	// Loop until connects
	for {
		if debug {
			log.Println("Worker Trying to conenct to manager")
		}

		conn, err := managerrequest.CreateWebsocket(config, config.CA, verifyAltName, verbose, debug)
		if err != nil {
			log.Println("Worker Error worker CreateWebsocket: ", err)
		} else {
			SetConn(conn, config, verbose, debug, writeLock)

			err = managerrequest.AddWorker(config, verbose, debug, writeLock)
			if err != nil {
				if verbose {
					log.Println("Worker Error worker AddWorker: ", err)
				}
			} else {
				if verbose {
					log.Println("Worker connected to manager. ")
				}
				break
			}
		}
		time.Sleep(time.Second * 5)
	}

	if debug {
		log.Println("Connection created successfully")
	}
}

func messageAddTask(config *utils.WorkerConfig, status *globalstructs.WorkerStatus, msg globalstructs.WebsocketMessage, verbose, debug bool, writeLock *sync.Mutex) (globalstructs.WebsocketMessage, error) {

	response := globalstructs.WebsocketMessage{
		Type: "",
		JSON: "",
	}

	if debug {
		log.Println("WebSockets msg.Type", msg.Type)
	}
	var requestTask globalstructs.Task
	err := json.Unmarshal([]byte(msg.JSON), &requestTask)
	if err != nil {
		return response, fmt.Errorf("WebSockets addWorker Unmarshal error: %s", err.Error())
	}
	// if executing task skip and return error
	if (config.DefaultThreads - len(status.WorkingIDs)) <= 0 {
		response.Type = "FAILED;addTask"
		response.JSON = msg.JSON

		requestTask.Status = "failed"
	} else {
		// Process task in background
		if debug {
			log.Println("WebSockets Task")
		}
		go process.Task(status, config, &requestTask, verbose, debug, writeLock)
		response.Type = "OK;addTask"
		response.JSON = msg.JSON
		requestTask.Status = "running"
	}

	//return task
	jsonData, err := json.Marshal(requestTask)
	if err != nil {
		return response, fmt.Errorf("WebSockets Marshal error: %s", err.Error())
	}
	response.JSON = string(jsonData)

	return response, nil
}

func messageDeleteTask(config *utils.WorkerConfig, status *globalstructs.WorkerStatus, msg globalstructs.WebsocketMessage, verbose, debug bool, writeLock *sync.Mutex) (globalstructs.WebsocketMessage, error) {
	response := globalstructs.WebsocketMessage{
		Type: "",
		JSON: "",
	}
	if debug {
		log.Println("WebSockets msg.Type", msg.Type)
	}

	var requestTask globalstructs.Task
	err := json.Unmarshal([]byte(msg.JSON), &requestTask)
	if err != nil {
		return response, fmt.Errorf("WebSockets deleteTask Unmarshal error: %s", err.Error())
	}

	cmdID := status.WorkingIDs[requestTask.ID]

	if cmdID < 0 {
		log.Println("Invalid cmdID")
		return response, fmt.Errorf("Invalid cmdID")
	}
	cmdIDString := strconv.Itoa(cmdID)

	// Kill the process using cmdID
	cmd := exec.Command("kill", "-9", cmdIDString)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()

	if err != nil {
		if debug {
			log.Println("WebSockets Error killing process:", err)
			log.Println("WebSockets Error details:", stderr.String())
		}
		response.Type = "FAILED;deleteTask"
		response.JSON = msg.JSON
	} else {
		response.Type = "OK;deleteTask"
		response.JSON = msg.JSON
	}

	return response, nil
}

func messageStatusTask(config *utils.WorkerConfig, status *globalstructs.WorkerStatus, msg globalstructs.WebsocketMessage, verbose, debug bool) (globalstructs.WebsocketMessage, error) {
	response := globalstructs.WebsocketMessage{
		Type: "",
		JSON: "",
	}
	status.IddleThreads = config.DefaultThreads - len(status.WorkingIDs)

	if debug {
		log.Println("WebSockets msg.Type", msg.Type, "status:", status)
	}

	jsonData, err := json.Marshal(status)
	if err != nil {
		response.Type = "FAILED"
	} else {
		response.Type = "status"
		response.JSON = string(jsonData)
	}

	if debug {
		// Print the JSON data
		log.Println("messageStatusTask:", string(jsonData))
	}
	return response, nil
}
