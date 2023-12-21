package modules

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/worker/utils"
)

var mutex sync.Mutex

func runModule(config *utils.WorkerConfig, command string, arguments string, status *globalstructs.WorkerStatus, id string, verbose, debug bool) (string, error) {
	// if command is empty, like in the example "exec" to exec any binary
	// the first argument is the command
	var cmd *exec.Cmd
	if config.InsecureModules {
		cmdStr := command + " " + arguments
		if debug {
			log.Println("cmdStr: ", cmdStr)
		}
		cmd = exec.Command("sh", "-c", cmdStr)

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
			log.Println("command: ", command)
			log.Println("argumentsArray: ", argumentsArray)
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
			fmt.Println("Standard Error:")
			fmt.Print(stderr.String())
		} else {
			// Some other error occurred
			fmt.Printf("Command finished with unexpected error: %v\n", err)
		}
		return err.Error(), err
	}

	mutex.Lock()
	status.WorkingIDs[id] = cmd.Process.Pid
	mutex.Unlock()

	// Wait for the command to finish
	err = cmd.Wait()
	if err != nil {
		if debug {
			log.Println("Error waiting for command:", err)
		}
		return err.Error(), err
	}

	// Capture the output of the script
	// Combine the standard output and standard error into a single string
	output := stdout.String() + stderr.String()
	output = strings.TrimRight(output, "\n")

	//Remove the ID from the status
	mutex.Lock()
	delete(status.WorkingIDs, id)
	mutex.Unlock()

	return output, nil
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

		// If there is a file in the command, save to disk
		if command.FileContent != "" {
			if command.RemoteFilePath == "" {
				return fmt.Errorf("RemoteFilePath empty")
			}

			err := SaveStringToFile(command.RemoteFilePath, command.FileContent)
			if err != nil {
				return err
			}

		}

		// Execute the module and get the output and any error
		outputCommand, err := runModule(config, commandAux, arguments, status, id, verbose, debug)
		if err != nil {
			// Return an error if there is an issue running the module
			return fmt.Errorf("error running task: %v", err)
		}

		// Store the output in the task struct for the current command
		task.Commands[num].Output = outputCommand
	}

	// Return nil if the task is processed successfully
	return nil
}

func GetRandomDuration(base, random int, verbose, debug bool) int {
	return rand.Intn(random) + base
}

func StringList(list []string, verbose, debug bool) string {
	stringList := ""
	for _, item := range list {
		stringList += item + "\n"
	}

	return stringList
}

// SaveStringToFile saves a string to a file.
func SaveStringToFile(filename string, content string) error {
	// Write the string content to the file
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("error saving string to file: %v", err)
	}

	fmt.Printf("String saved to file: %s\n", filename)
	return nil
}
