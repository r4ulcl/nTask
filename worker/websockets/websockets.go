package websockets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/worker/managerRequest"
	"github.com/r4ulcl/nTask/worker/process"
	"github.com/r4ulcl/nTask/worker/utils"
)

func GetMessage(config *utils.WorkerConfig, status *globalstructs.WorkerStatus, verbose, debug bool, writeLock *sync.Mutex) {
	for {

		response := globalstructs.WebsocketMessage{
			Type: "",
			Json: "",
		}

		_, p, err := config.Conn.ReadMessage() //messageType
		if err != nil {
			log.Println("config.Conn.ReadMessage()", err)
			time.Sleep(time.Second * 5)
			continue
		}

		var msg globalstructs.WebsocketMessage
		err = json.Unmarshal(p, &msg)
		if err != nil {
			log.Println("Error decoding JSON:", err)
			continue
		}

		switch msg.Type {

		case "status":
			if debug {
				if debug {
					log.Println("msg.Type", msg.Type)
				}
			}
			jsonData, err := json.Marshal(status)
			if err != nil {
				response.Type = "FAILED"
			} else {
				response.Type = "status"
				response.Json = string(jsonData)
			}

			if debug {
				// Print the JSON data
				log.Println(string(jsonData))
			}

		case "addTask":
			if debug {
				log.Println("msg.Type", msg.Type)
			}
			var requestTask globalstructs.Task
			err = json.Unmarshal([]byte(msg.Json), &requestTask)
			if err != nil {
				log.Println("addWorker Unmarshal error: ", err)
			}
			// if executing task skip and return error
			if status.IddleThreads <= 0 {
				response.Type = "FAILED;addTask"
				response.Json = msg.Json

				requestTask.Status = "failed"
			} else {
				// Process task in background
				if debug {
					log.Println("ProcessTask")
				}
				go process.ProcessTask(status, config, &requestTask, verbose, debug, writeLock)
				response.Type = "OK;addTask"
				response.Json = msg.Json
				requestTask.Status = "running"
			}

			//return task
			jsonData, err := json.Marshal(requestTask)
			if err != nil {
				log.Println("Marshal error: ", err)
			}
			response.Json = string(jsonData)

		case "deleteTask":
			if debug {
				log.Println("msg.Type", msg.Type)
			}

			var requestTask globalstructs.Task
			err = json.Unmarshal([]byte(msg.Json), &requestTask)
			if err != nil {
				log.Println("deleteTask Unmarshal error: ", err)
			}

			cmdID := status.WorkingIDs[requestTask.ID]
			// Kill the process using cmdID
			cmd := exec.Command("kill", "-9", fmt.Sprint(cmdID))

			var stderr bytes.Buffer
			cmd.Stderr = &stderr

			err := cmd.Run()

			if err != nil {
				if debug {
					log.Println("Error killing process:", err)
					log.Println("Error details:", stderr.String())
				}
				response.Type = "FAILED;deleteTask"
				response.Json = msg.Json
			} else {
				response.Type = "OK;deleteTask"
				response.Json = msg.Json
			}
		}
		if debug {
			log.Printf("Received message type: %s\n", msg.Type)
			log.Printf("Received message json: %s\n", msg.Json)
		}

		if response.Type != "" {
			jsonData, err := json.Marshal(response)
			if err != nil {
				log.Println("Marshal error: ", err)
			}
			err = managerRequest.SendMessage(config.Conn, jsonData, verbose, debug, writeLock)
			if err != nil {
				log.Println("SendMessage error: ", err)
			}
		}
	}
}

func RecreateConnection(config *utils.WorkerConfig, verifyAltName, verbose, debug bool, writeLock *sync.Mutex) {
	for {
		time.Sleep(1 * time.Second) // Adjust the interval based on your requirements
		if err := config.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(1*time.Second)); err != nil {
			conn, err := managerRequest.CreateWebsocket(config, config.CA, verifyAltName, verbose, debug)
			if err != nil {
				if verbose {
					log.Println("Error CreateWebsocket: ", err)
				}
			} else {
				config.Conn = conn

				err = managerRequest.AddWorker(config, verbose, debug, writeLock)
				if err != nil {
					if verbose {
						log.Println("Error worker: ", err)
					}
				} else {
					if verbose {
						log.Println("Worker connected to manager. ")
					}
					continue
				}

			}
		}
	}
}
