package worker

import (
	"fmt"
	"os/exec"
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
