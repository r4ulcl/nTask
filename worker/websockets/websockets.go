package websockets

import (
	"bytes"
	"encoding/json"
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
				if debug {
					log.Println("WebSockets msg.Type", msg.Type)
				}
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
				log.Println(string(jsonData))
			}

		case "addTask":
			if debug {
				log.Println("WebSockets msg.Type", msg.Type)
			}
			var requestTask globalstructs.Task
			err = json.Unmarshal([]byte(msg.JSON), &requestTask)
			if err != nil {
				log.Println("WebSockets addWorker Unmarshal error: ", err)
			}
			// if executing task skip and return error
			if status.IddleThreads <= 0 {
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
				log.Println("WebSockets Marshal error: ", err)
			}
			response.JSON = string(jsonData)

		case "deleteTask":
			if debug {
				log.Println("WebSockets msg.Type", msg.Type)
			}

			var requestTask globalstructs.Task
			err = json.Unmarshal([]byte(msg.JSON), &requestTask)
			if err != nil {
				log.Println("WebSockets deleteTask Unmarshal error: ", err)
			}

			cmdID := status.WorkingIDs[requestTask.ID]

			if cmdID < 0 {
				log.Println("Invalid cmdID")
				continue
			}
			cmdIDString := strconv.Itoa(cmdID)

			// Kill the process using cmdID
			cmd := exec.Command("kill", "-9", cmdIDString)

			var stderr bytes.Buffer
			cmd.Stderr = &stderr

			err := cmd.Run()

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
	for {
		time.Sleep(1 * time.Second) // Adjust the interval based on your requirements
		if err := config.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(1*time.Second)); err != nil {
			conn, err := managerrequest.CreateWebsocket(config, config.CA, verifyAltName, verbose, debug)
			if err != nil {
				if verbose {
					log.Println("WebSockets Error CreateWebsocket: ", err)
				}
			} else {
				config.Conn = conn

				err = managerrequest.AddWorker(config, verbose, debug, writeLock)
				if err != nil {
					if verbose {
						log.Println("WebSockets Error worker RecreateConnection AddWorker: ", err)
					}
				} else {
					if verbose {
						log.Println("WebSockets Worker connected to manager. ")
					}

					continue
				}

			}
		}
	}
}
