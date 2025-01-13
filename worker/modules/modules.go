package modules

import (
	"bytes"
	"context"
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

	defer cleanupWorkerStatus(status, id)

	cmd, err := prepareCommand(config, command, arguments, debug)
	if err != nil {
		return "", err
	}

	output, err := executeCommand(cmd, status, id, verbose, debug)
	return strings.TrimRight(output, "\n"), err
}

func cleanupWorkerStatus(status *globalstructs.WorkerStatus, id string) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(status.WorkingIDs, id)
}

func prepareCommand(config *utils.WorkerConfig, command, arguments string, debug bool) (*exec.Cmd, error) {
	var cmd *exec.Cmd

	if config.InsecureModules {
		cmd = createInsecureCommand(command, arguments, debug)
	} else {
		var err error
		cmd, err = createSecureCommand(command, arguments, debug)
		if err != nil {
			return nil, err
		}
	}

	return cmd, nil
}

func createInsecureCommand(command, arguments string, debug bool) *exec.Cmd {
	cmdStr := command + " " + arguments
	if debug {
		log.Println("Modules cmdStr: ", cmdStr)
	}

	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", cmdStr)
	} else if runtime.GOOS == "linux" {
		// use --login to load bashrc
		return exec.Command("bash", "--login", "-c", cmdStr)
	}

	log.Fatal("Unsupported operating system")
	return nil
}

func createSecureCommand(command, arguments string, debug bool) (*exec.Cmd, error) {
	argumentsArray := strings.Split(arguments, " ")
	if command == "" && len(arguments) > 0 {
		command = argumentsArray[0]
		argumentsArray = argumentsArray[1:]
	}

	if strings.Contains(command, " ") {
		parts := strings.SplitN(command, " ", 2)
		argumentsArray = append([]string{parts[1]}, argumentsArray...)
		command = parts[0]
	}

	if debug {
		log.Println("Modules command: ", command)
		log.Println("Modules argumentsArray: ", argumentsArray)
	}

	return exec.Command(command, argumentsArray...), nil
}

func executeCommand(cmd *exec.Cmd, status *globalstructs.WorkerStatus, id string, verbose, debug bool) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		logCommandError(err, &stderr, verbose, debug)
		return "", err
	}

	mutex.Lock()
	status.WorkingIDs[id] = cmd.Process.Pid
	mutex.Unlock()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	return monitorCommandExecution(cmd, &stdout, &stderr, done, verbose, debug)
}

func logCommandError(err error, stderr *bytes.Buffer, verbose, debug bool) {
	if exitError, ok := err.(*exec.ExitError); ok {
		if verbose || debug {
			log.Printf("Command exited with error: %v", exitError)
			log.Println("Standard Error:")
			log.Print(stderr.String())
		}
	} else {
		if verbose || debug {
			log.Printf("Command finished with unexpected error: %v", err)
		}
	}
}

func monitorCommandExecution(cmd *exec.Cmd, stdout, stderr *bytes.Buffer, done chan error, verbose, debug bool) (string, error) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := isProcessRunning(cmd.Process.Pid, verbose, debug); err != nil {
				return stdout.String() + stderr.String(), err
			}
		case err := <-done:
			if err != nil && debug {
				log.Println("Modules Error waiting for command:", err)
			}
			return stdout.String() + stderr.String(), err
		}
	}
}

// Function to check if a process with a given PID is still running
func isProcessRunning(pid int, verbose, debug bool) error {
	if debug || verbose {
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

// DeleteFiles delete files send
func DeleteFiles(task *globalstructs.Task, verbose, debug bool) error {
	for num, file := range task.Files {
		path := file.RemoteFilePath

		// Attempt to delete the file
		err := os.Remove(path)
		if err != nil {
			fmt.Printf("Error deleting file: %v\n", err)
		}

		// Verbose logging
		if verbose {
			fmt.Printf("Deleted file %d to %s\n", num+1, path)
		}

		// Debug logging
		if debug {
			fmt.Printf("Debug: File %d details - Path: %s, Content\n", num+1, path)
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
	// Define a context with timeout for the entire task
	var ctx context.Context
	var cancel context.CancelFunc
	if task.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(task.Timeout)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	// Channel to signal a timeout or completion
	done := make(chan error, 1)

	// Run the task processing in a separate goroutine
	go func() {
		// Start timer to measure the command execution time
		startTime := time.Now()
		for num, command := range task.Commands {
			module := command.Module
			arguments := command.Args

			// Check if the module exists in the worker configuration
			commandAux, found := config.Modules[module]
			if !found {
				// Send an error if the module is not found
				done <- fmt.Errorf("unknown command: %s", module)
				return
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
				// Send an error if there is an issue running the module
				done <- fmt.Errorf("error running %s task: %v", commandAux, err)
				return
			}

			// Store the output in the task struct for the current command
			task.Commands[num].Output = outputCommand
		}
		// Calculate and save the duration in seconds
		duration := time.Since(startTime).Seconds()
		task.Duration = duration

		// Signal successful completion
		done <- nil
	}()

	// Wait for the task to complete or timeout
	select {
	case err := <-done:
		if err != nil {
			return err
		}
		// Return nil if the task is processed successfully
		return nil
	case <-ctx.Done():
		// Set a timeout error for all commands if the context times out
		for i := range task.Commands {
			task.Commands[i].Output = "Timeout error: task exceeded the time limit"
		}
		return fmt.Errorf("timeout processing task: exceeded %d seconds", task.Timeout)
	}
}

func stringList(list []string, verbose, debug bool) string {
	if verbose || debug {
		log.Println("Executing stringList")
		if debug {
			log.Println("    with params:", list)

		}
	}
	stringList := ""
	for _, item := range list {
		stringList += item + "\n"
	}

	return stringList
}

// saveStringToFile saves a string to a file.
func saveStringToFile(filename string, content string) error {
	// Write the string content to the file
	err := os.WriteFile(filename, []byte(content), 0600)
	if err != nil {
		return fmt.Errorf("error saving string to file: %v", err)
	}

	fmt.Printf("String saved to file: %s\n", filename)
	return nil
}
