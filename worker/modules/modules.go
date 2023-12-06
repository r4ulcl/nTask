package modules

import (
	"fmt"
	"log"
	"math/rand"
	"os/exec"
	"strings"

	"github.com/r4ulcl/NetTask/globalstructs"
	"github.com/r4ulcl/NetTask/worker/utils"
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

	log.Println("command: ", command)
	log.Println("arguments: ", arguments)

	// Command to run the module
	cmd := exec.Command(command, arguments...)

	// Start the command
	err := cmd.Start()
	if err != nil {
		fmt.Println("Error starting command:", err)
		return "", err
	}

	status.WorkingIDs[id] = cmd.Process.Pid

	// Wait for the command to finish
	err = cmd.Wait()
	if err != nil {
		fmt.Println("Error waiting for command:", err)
	}

	// Capture the output of the script
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	// Convert the output byte slice to a string
	outputString := string(output)

	//Remove the ID from the status
	delete(status.WorkingIDs, id)

	return outputString, nil
}

func ProcessModule(task *globalstructs.Task, config *utils.WorkerConfig, status *globalstructs.WorkerStatus, id string, verbose bool) (string, error) {
	module := task.Module
	arguments := task.Args

	command, found := config.Modules[module]
	if !found {
		return "Unknown task", fmt.Errorf("unknown command")
	}

	return runModule(command, arguments, status, id, verbose)
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
