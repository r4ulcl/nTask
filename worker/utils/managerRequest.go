package utils

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	globalstructs "github.com/r4ulcl/nTask/globalstructs"
)

func CreateWebsocket(config *WorkerConfig, verbose, debug bool) (*websocket.Conn, error) {
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
func AddWorker(config *WorkerConfig, verbose, debug bool, writeLock *sync.Mutex) error {
	// Create a Worker object with the provided configuration
	worker := globalstructs.Worker{
		Name:         config.Name,
		Port:         config.Port,
		OauthToken:   config.OAuthToken,
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

	/*
		// Create a new POST request to add the worker
		var url string
		if transport, ok := config.ClientHTTP.Transport.(*http.Transport); ok {
			if transport.TLSClientConfig != nil {
				url = "https://" + config.ManagerIP + ":" + config.ManagerPort + "/worker"
			} else {
				url = "http://" + config.ManagerIP + ":" + config.ManagerPort + "/worker"
			}
		} else {
			url = "http://" + config.ManagerIP + ":" + config.ManagerPort + "/worker"
		}
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		if err != nil {
			return err
		}

		// Add custom headers, including the OAuth header
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", config.ManagerOauthToken)

		// Create an HTTP client and make the request
		resp, err := config.ClientHTTP.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		const maxBufferSize = 1024
		buffer := make([]byte, maxBufferSize)
		// Use a bytes.Buffer to accumulate the response body
		var body bytes.Buffer

		// Use io.Copy to efficiently copy the response body to the buffer
		_, err = io.CopyBuffer(&body, resp.Body, buffer)
		if err != nil {
			fmt.Println("Error copying response body:", err)
			return err
		}

		// Check if the response status code is not 200
		if resp.StatusCode != 200 {
			return fmt.Errorf("error adding the worker %d", body)
		}*/

	return nil
}

// AddWorker sends a POST request to add a worker to the manager
func DeleteWorker(config *WorkerConfig, verbose, debug bool, writeLock *sync.Mutex) error {
	// Create a Worker object with the provided configuration
	worker := globalstructs.Worker{
		Name:         config.Name,
		Port:         config.Port,
		OauthToken:   config.OAuthToken,
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

	/*
		var url string
		if transport, ok := config.ClientHTTP.Transport.(*http.Transport); ok {
			if transport.TLSClientConfig != nil {
				url = "https://" + config.ManagerIP + ":" + config.ManagerPort + "/worker/" + config.Name
			} else {
				url = "http://" + config.ManagerIP + ":" + config.ManagerPort + "/worker/" + config.Name
			}
		} else {
			url = "http://" + config.ManagerIP + ":" + config.ManagerPort + "/worker/" + config.Name
		}

		// Create a new DELETE request
		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			return err
		}

		// Add custom headers, including the OAuth header
		req.Header.Set("Authorization", config.ManagerOauthToken)

		// Specify the content type as JSON
		req.Header.Set("Content-Type", "application/json")

		// Send the request

		resp, err := config.ClientHTTP.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Check the response status
		log.Println(resp.StatusCode)
		if resp.StatusCode == http.StatusOK {
			if debug {
				log.Println("DELETE request was successful")
			}
		}
	*/
	return nil
}

// CallbackTaskMessage sends a POST request to the manager to callback with a task message
func CallbackTaskMessage(config *WorkerConfig, task *globalstructs.Task, verbose, debug bool, writeLock *sync.Mutex) error {
	// Marshal the task object into JSON
	payload, _ := json.Marshal(task)

	msg := globalstructs.WebsocketMessage{
		Type: "callbackTask",
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

	/*
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
			log.Println("Response callbackTask OK from manager")
		} else {
			log.Println("Response callbackTask not OK from manager", msg.Type)
		}*/
	/*
		// Create the callback URL using the manager IP and port
		var url string
		if transport, ok := config.ClientHTTP.Transport.(*http.Transport); ok {
			if transport.TLSClientConfig != nil {
				url = "https://" + config.ManagerIP + ":" + config.ManagerPort + "/callback"
			} else {
				url = "http://" + config.ManagerIP + ":" + config.ManagerPort + "/callback"
			}
		} else {
			url = "http://" + config.ManagerIP + ":" + config.ManagerPort + "/callback"
		}
		// Marshal the task object into JSON
		payload, _ := json.Marshal(task)

		// Create a new POST request to send the task message
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		if err != nil {
			if debug {
				log.Println("Error creating request:", err)
			}
			return err
		}

		// Add custom headers, including the OAuth header
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", config.ManagerOauthToken)

		// Create an HTTP client and make the request

		resp, err := config.ClientHTTP.Do(req)
		if err != nil {
			if debug {
				log.Println("Error making request:", err)
			}
			return err
		}
		defer resp.Body.Close()

		if debug {
			log.Println("Status Code:", resp.Status)
		}
		// Handle the response body as needed
	*/
	return nil
}
