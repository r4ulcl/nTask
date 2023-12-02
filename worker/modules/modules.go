package modules

import (
	"log"
	"math/rand"
	"os/exec"
	"time"
)

func Module1(arguments []string) (string, error) {
	// Command to run the Python script
	scriptPath := "./worker/modules/module1.py"
	cmd := exec.Command("python3", append([]string{scriptPath}, arguments...)...)

	// Capture the output of the script
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	// Convert the output byte slice to a string
	outputString := string(output)

	return outputString, nil
}

func Module2(arguments []string) (string, error) {
	// Command to run the Bash script
	scriptPath := "./worker/modules/module2.sh"
	cmd := exec.Command("bash", append([]string{scriptPath}, arguments...)...)

	// Capture the output of the script
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	// Convert the output byte slice to a string
	outputString := string(output)

	return outputString, nil
}

func WorkAndNotify(id string) (string, error) {
	// workMutex.Lock()
	// isWorking = true
	// messageID = id
	// workMutex.Unlock()

	// Simulate work with an unknown duration
	workDuration := GetRandomDuration()
	log.Println("Working for ", workDuration.String(), " ID: ", id)
	time.Sleep(workDuration)

	// workMutex.Lock()
	// isWorking = false
	// messageID = ""
	// workMutex.Unlock()
	str := "Working for " + workDuration.String() + " (ID: " + id + ")"
	return str, nil
}

func GetRandomDuration() time.Duration {
	return time.Duration(rand.Intn(10)+1) * time.Second
}

func StringList(list []string) string {
	stringList := ""
	for _, item := range list {
		stringList += item + "\n"
	}

	return stringList
}
