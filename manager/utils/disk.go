package utils

import (
	"encoding/json"
	"log"
	"os"

	"github.com/r4ulcl/nTask/globalstructs"
)

// SaveTaskToDisk Save Task To Disk
func SaveTaskToDisk(task globalstructs.Task, path string, verbose, debug bool) error {
	// Convert the struct to JSON format
	jsonData, err := json.MarshalIndent(task, "", "    ")
	if err != nil {
		if verbose {
			log.Println("Utils Error marshaling JSON:", err)
		}
		return err
	}

	// Get date and time
	//currentTime := time.Now()
	// Specify the file path
	//	filePath := path + "/" + task.ID + ".json"
	//filePath := fmt.Sprintf("%s/%s_%s.json", path, currentTime.Format("2006-01-02_15-04-05"), task.ID)
	filePath := path

	// Open the file for writing
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		if verbose {
			log.Println("Error creating file:", err)
		}
		return err
	}
	defer file.Close()

	// Write the JSON data to the file
	_, err = file.Write(jsonData)
	if err != nil {
		if verbose {
			log.Println("Error writing to file:", err)
		}
		return err
	}
	return nil
}
