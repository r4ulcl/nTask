package modules

import (
	"fmt"
	"math/rand"
	"os/exec"

	"github.com/r4ulcl/NetTask/globalstructs"
	"github.com/r4ulcl/NetTask/worker/utils"
)

func runModule(command string, arguments []string) (string, error) {
	// Command to run the Python script
	cmd := exec.Command(command, arguments...)

	// Capture the output of the script
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	// Convert the output byte slice to a string
	outputString := string(output)

	return outputString, nil
}

func ProcessModule(task *globalstructs.Task, config *utils.WorkerConfig) (string, error) {
	module := task.Module
	arguments := task.Args

	command, found := config.Modules[module]
	if !found {
		return "Unknown task", fmt.Errorf("unknown command")
	}

	return runModule(command, arguments)
}

func GetRandomDuration(base, random int) int {
	return rand.Intn(random) + base
}

func StringList(list []string) string {
	stringList := ""
	for _, item := range list {
		stringList += item + "\n"
	}

	return stringList
}
