package worker

import (
	"fmt"
	"math/rand"
	"os/exec"
	"time"
)

func module1(arguments []string) (string, int) {
	// Command to run the Python script
	scriptPath := "./worker/modules/module1.py"
	cmd := exec.Command("python3", append([]string{scriptPath}, arguments...)...)

	// Capture the output of the script
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Convert the output byte slice to a string
	outputString := string(output)

	return outputString, 0
}

func module2(arguments []string) (string, int) {
	// Command to run the Bash script
	scriptPath := "./worker/modules/module2.sh"
	cmd := exec.Command("bash", append([]string{scriptPath}, arguments...)...)

	// Capture the output of the script
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Convert the output byte slice to a string
	outputString := string(output)

	return outputString, 0
}

func workAndNotify(seconds int, id string) {
	//workMutex.Lock()
	Working = true
	messageID = id
	//workMutex.Unlock()

	// Simulate work with an unknown duration
	workDuration := getRandomDuration()
	fmt.Printf("Working for %s (ID: %s)\n", workDuration.String(), id)
	time.Sleep(workDuration)

	//workMutex.Lock()
	Working = false
	messageID = ""
	//workMutex.Unlock()
}

func getRandomDuration() time.Duration {
	return time.Duration(rand.Intn(10)+1) * time.Second
}
