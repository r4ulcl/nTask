package websockets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
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

func initConnDeadlines(c *websocket.Conn) {
	c.SetReadLimit(globalstructs.MaxMessageSize)
	c.SetReadDeadline(time.Now().Add(globalstructs.PongWait))
}

// attachPongHandler resets deadlines and signals when a Pong arrives
func attachPongHandler(conn *websocket.Conn, pongRec chan struct{}, debug bool) {
	conn.SetPongHandler(func(appData string) error {
		if debug {
			log.Println("Received Pong:", appData)
		}
		conn.SetReadDeadline(time.Now().Add(globalstructs.PongWait))
		// non-blocking notify
		select {
		case pongRec <- struct{}{}:
		default:
		}
		return nil
	})
}

func GetMessage(config *utils.WorkerConfig, status *globalstructs.WorkerStatus, verbose, debug bool, writeLock *sync.Mutex) {
	for {
		// blocking read → any error bubbles up
		_, p, err := config.Conn.ReadMessage()
		if err != nil {
			log.Println("Conn.ReadMessage error:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var msg globalstructs.WebsocketMessage
		if err := json.Unmarshal(p, &msg); err != nil {
			log.Println("JSON decode error:", err)
			continue
		}

		if debug {
			log.Printf("Received message type=%q json=%s\n", msg.Type, msg.JSON)
		}

		var (
			response   globalstructs.WebsocketMessage
			handlerErr error
		)
		switch msg.Type {
		case "status":
			response, handlerErr = messageStatusTask(config, status, msg, verbose, debug)
		case "addTask":
			response, handlerErr = messageAddTask(config, status, msg, verbose, debug, writeLock)
		case "deleteTask":
			response, handlerErr = messageDeleteTask(status, msg, verbose, debug)
		default:
			if debug {
				log.Printf("Unhandled message type: %s", msg.Type)
			}
		}
		if handlerErr != nil {
			log.Println("Handler error:", handlerErr)
		}

		if response.Type != "" {
			jsonData, _ := json.Marshal(response)
			if err := managerrequest.SendMessage(config.Conn, jsonData, verbose, debug, writeLock); err != nil {
				log.Println("SendMessage error:", err)
			}
		}
	}
}

// RecreateConnection keeps the connection healthy: it sends pings every 5 s and
// reconnects if a Pong is not received within 5 s.
func RecreateConnection(config *utils.WorkerConfig, verifyAltName, verbose, debug bool, writeLock *sync.Mutex) {
	pongReceived := make(chan struct{}, 1)
	// ensure handler on first conn
	attachPongHandler(config.Conn, pongReceived, debug)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if debug {
			log.Println("Heartbeat check")
		}
		// send Ping under lock
		writeLock.Lock()
		err := config.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
		writeLock.Unlock()
		if err != nil {
			log.Println("Error sending Ping, reconnecting:", err)
			config.Conn.Close()
			CreateConnection(config, verifyAltName, verbose, debug, writeLock)
			attachPongHandler(config.Conn, pongReceived, debug)
			continue
		}

		// wait for Pong or timeout
		timeout := time.NewTimer(5 * time.Second)
		select {
		case <-pongReceived:
			if debug {
				log.Println("pongReceived – connection healthy")
			}
		case <-timeout.C:
			log.Println("Pong timeout – reconnecting")
			config.Conn.Close()
			CreateConnection(config, verifyAltName, verbose, debug, writeLock)
			attachPongHandler(config.Conn, pongReceived, debug)
		}
		timeout.Stop()
	}
}

// CreateConnection dials the manager, stores it in config.Conn, installs
// deadlines, and registers the worker.
func CreateConnection(config *utils.WorkerConfig, verifyAltName, verbose, debug bool, writeLock *sync.Mutex) {
	for {
		if debug {
			log.Println("Attempting to connect...")
		}
		conn, err := managerrequest.CreateWebsocket(config, config.CA, verifyAltName, verbose, debug)
		if err != nil {
			log.Println("CreateWebsocket error:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		initConnDeadlines(conn)
		writeLock.Lock()
		config.Conn = conn
		writeLock.Unlock()

		// optional TCP keep-alive
		if tcpConn, ok := conn.UnderlyingConn().(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)
		}

		if err := managerrequest.AddWorker(config, verbose, debug, writeLock); err != nil {
			log.Println("AddWorker error:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if verbose {
			log.Println("Connected to manager ✓")
		}
		return
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

func messageDeleteTask(status *globalstructs.WorkerStatus, msg globalstructs.WebsocketMessage, verbose, debug bool) (globalstructs.WebsocketMessage, error) {
	response := globalstructs.WebsocketMessage{
		Type: "",
		JSON: "",
	}
	if debug || verbose {
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

	if debug || verbose {
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
