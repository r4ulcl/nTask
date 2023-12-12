package modules

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os/exec"
	"strings"

	"github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/worker/utils"
)

func runModule(command string, arguments []string, status *globalstructs.WorkerStatus, id string, verbose bool) (string, error) {
	// if command is empty, like in the example "exec" to exec any binary
	// the first argument is the command
	if command == "" && len(arguments) > 0 {
		command = arguments[0]
		arguments = arguments[1:]
	}

	// Check if module has space, to separate it in command and args
	if strings.Contains(command, " ") {
		parts := strings.SplitN(command, " ", 2)
		arguments = append([]string{parts[1]}, arguments...)

		// Update the inputString to contain only the first part
		command = parts[0]
	}

	if verbose {
		log.Println("command: ", command)
		log.Println("arguments: ", arguments)
	}

	// Command to run the module
	cmd := exec.Command(command, arguments...)

	// Create a buffer to store the command output
	var stdout, stderr bytes.Buffer

	// Set the output and error streams to the buffers
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command
	err := cmd.Start()
	if err != nil {
		if verbose {
			fmt.Println("Error starting command:", err)
		}
		return "", err
	}

	status.WorkingIDs[id] = cmd.Process.Pid

	// Wait for the command to finish
	err = cmd.Wait()
	if err != nil {
		if verbose {
			fmt.Println("Error waiting for command:", err)
		}
		return "", err
	}

	// Capture the output of the script
	// Combine the standard output and standard error into a single string
	output := stdout.String() + stderr.String()
	output = strings.TrimRight(output, "\n")

	//Remove the ID from the status
	delete(status.WorkingIDs, id)

	return output, nil
}

// ProcessModule processes a task by iterating through its commands and executing corresponding modules
func ProcessModule(task *globalstructs.Task, config *utils.WorkerConfig, status *globalstructs.WorkerStatus, id string, verbose bool) error {
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
		outputCommand, err := runModule(commandAux, arguments, status, id, verbose)
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

func GetRandomDuration(base, random int, verbose bool) int {
	return rand.Intn(random) + base
}

func StringList(list []string, verbose bool) string {
	stringList := ""
	for _, item := range list {
		stringList += item + "\n"
	}

	return stringList
}

// SaveStringToFile saves a string to a file.
func SaveStringToFile(filename string, content string) error {
	// Write the string content to the file
	err := ioutil.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("error saving string to file: %v", err)
	}

	fmt.Printf("String saved to file: %s\n", filename)
	return nil
}
