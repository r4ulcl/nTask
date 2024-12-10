package modules

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/worker/utils"
)

var mutex sync.Mutex

func runModule(config *utils.WorkerConfig, command string, arguments string, status *globalstructs.WorkerStatus, id string, verbose, debug bool) (string, error) {
	mutex.Lock()
	status.WorkingIDs[id] = -1
	mutex.Unlock()

	defer func() {
		mutex.Lock()
		delete(status.WorkingIDs, id)
		mutex.Unlock()
	}()

	// if command is empty, like in the example "exec" to exec any binary
	// the first argument is the command
	var cmd *exec.Cmd
	if config.InsecureModules {
		cmdStr := command + " " + arguments
		if debug {
			log.Println("Modules cmdStr: ", cmdStr)
		}

		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", cmdStr)
		} else if runtime.GOOS == "linux" {
			cmd = exec.Command("sh", "-c", cmdStr)
		} else {
			log.Fatal("Unsupported operating system")
		}

	} else {
		// Convert arguments to array
		argumentsArray := strings.Split(arguments, " ")
		if command == "" && len(arguments) > 0 {
			command = argumentsArray[0]
			argumentsArray = argumentsArray[1:]
		}

		// Check if module has space, to separate it in command and args
		if strings.Contains(command, " ") {
			parts := strings.SplitN(command, " ", 2)
			argumentsArray = append([]string{parts[1]}, argumentsArray...)

			// Update the inputString to contain only the first part
			command = parts[0]
		}

		if debug {
			log.Println("Modules command: ", command)
			log.Println("Modules argumentsArray: ", argumentsArray)
		}

		// Command to run the module
		cmd = exec.Command(command, argumentsArray...)
	}
	// Create a buffer to store the command output
	var stdout, stderr bytes.Buffer

	// Set the output and error streams to the buffers
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command
	err := cmd.Start()
	if err != nil {
		// Check if the error is an ExitError
		if exitError, ok := err.(*exec.ExitError); ok {
			// The command exited with a non-zero status
			fmt.Printf("Command exited with error: %v\n", exitError)

			// Print the captured standard error
			log.Println("Standard Error:")
			fmt.Print(stderr.String())
		} else {
			// Some other error occurred
			fmt.Printf("Command finished with unexpected error: %v\n", err)
		}
		return "", err
	}

	mutex.Lock()
	status.WorkingIDs[id] = cmd.Process.Pid
	mutex.Unlock()

	// Create a channel to signal when the process is done
	done := make(chan error, 1)

	// Monitor the process in a goroutine
	go func() {
		// Wait for the command to finish
		err := cmd.Wait()
		done <- err
	}()

	// Check every 30 minutes if the process is still running
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if the process is still running
			if err := isProcessRunning(cmd.Process.Pid, verbose, debug); err != nil {
				// Process is not running, break the loop
				output := stdout.String() + stderr.String()
				output = strings.TrimRight(output, "\n")
				return output, err
			}
		case err := <-done:
			// Process has finished
			if err != nil {
				if debug {
					log.Println("Modules Error waiting for command:", err)
				}
				output := stdout.String() + stderr.String()
				output = strings.TrimRight(output, "\n")
				return output, err
			}

			// Process completed successfully
			// Capture the output of the script
			output := stdout.String() + stderr.String()
			output = strings.TrimRight(output, "\n")
			return output, nil
		}
	}
}

// Function to check if a process with a given PID is still running
func isProcessRunning(pid int, verbose, debug bool) error {
	if debug {
		log.Println("isProcessRunning", pid)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	// Send a signal of 0 to check if the process exists
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process does not exist
		return fmt.Errorf("Process with PID %d is not running", pid)
	}

	// Process is still running
	return nil
}

// ProcessFiles decodes the base64 content of each file in task.Files and saves it to its RemoteFilePath.
// It updates the WorkerStatus and handles verbose and debug logging as needed.
func ProcessFiles(task *globalstructs.Task, config *utils.WorkerConfig, status *globalstructs.WorkerStatus, id string, verbose, debug bool) error {
	for num, file := range task.Files {
		// Assuming 'fileContentB64B64' is the base64-encoded content as a string
		// If this is a typo, rename it appropriately (e.g., 'FileContentB64')
		contentB64 := file.FileContentB64
		path := file.RemoteFilePath

		// Decode the base64 content
		decodedBytes, err := base64.StdEncoding.DecodeString(contentB64)
		if err != nil {
			return fmt.Errorf("file %d: failed to decode base64 content: %w", num+1, err)
		}

		// Ensure the directory exists
		dir := getDirectory(path)
		const dirPerm = 0600 // Use restricted permissions (0600)
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return fmt.Errorf("file %d: failed to create directories for %s: %w", num+1, path, err)
		}
		// Write the decoded content to the specified path
		const filePerm = 0600
		if err := os.WriteFile(path, decodedBytes, filePerm); err != nil {
			return fmt.Errorf("file %d: failed to write file %s: %w", num+1, path, err)
		}

		// Update the worker status if applicable
		// (Assuming WorkerStatus has a method or field to update progress)
		// status.UpdateProgress(num + 1, len(task.Files))

		// Verbose logging
		if verbose {
			fmt.Printf("Saved file %d to %s\n", num+1, path)
		}

		// Debug logging
		if debug {
			fmt.Printf("Debug: File %d details - Path: %s, Content Length: %d bytes\n", num+1, path, len(decodedBytes))
		}
	}

	return nil
}

// getDirectory extracts the directory part from a file path using filepath.Dir
func getDirectory(filePath string) string {
	return filepath.Dir(filePath)
}

// ProcessModule processes a task by iterating through its commands and executing corresponding modules
func ProcessModule(task *globalstructs.Task, config *utils.WorkerConfig, status *globalstructs.WorkerStatus, id string, verbose, debug bool) error {
	for num, command := range task.Commands {
		module := command.Module
		arguments := command.Args

		// Check if the module exists in the worker configuration
		commandAux, found := config.Modules[module]
		if !found {
			// Return an error if the module is not found
			return fmt.Errorf("unknown command: %s", module)
		}

		if verbose {
			log.Println("Modules commandAux: ", commandAux)
			log.Println("Modules arguments: ", arguments)
		}

		// Execute the module and get the output and any error
		outputCommand, err := runModule(config, commandAux, arguments, status, id, verbose, debug)
		if err != nil {
			// Save the text error in the task output to review
			task.Commands[num].Output = outputCommand + ";" + err.Error()
			// Return an error if there is an issue running the module
			return fmt.Errorf("error running %s task: %v", commandAux, err)
		}

		// Store the output in the task struct for the current command
		task.Commands[num].Output = outputCommand
	}

	// Return nil if the task is processed successfully
	return nil
}

func stringList(list []string, verbose, debug bool) string {
	stringList := ""
	for _, item := range list {
		stringList += item + "\n"
	}

	return stringList
}

// SaveStringToFile saves a string to a file.
func SaveStringToFile(filename string, content string) error {
	// Write the string content to the file
	err := os.WriteFile(filename, []byte(content), 0600)
	if err != nil {
		return fmt.Errorf("error saving string to file: %v", err)
	}

	fmt.Printf("String saved to file: %s\n", filename)
	return nil
}
