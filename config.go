package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

const configPath = "config.json"
const failPath = "C:\\"

var configuration Configuration

// Configuration stores all our user defined variables
type Configuration struct {
	LogLevel      int    `json:"LogLevel"`      // 0=Off,1=Error,2=Warn,3=Info,4=Debug
	LogPath       string `json:"LogPath"`       // Where to write logs to
	EQLogPath     string `json:"EQLogPath"`     // Where to read logs from
	EQBaseLogLine string `json:"EQBaseLogLine"` // Regex for a eq log line
	TickAlert     int    `json:"TickAlert"`     // Remaining ticks to issue an alert at
	ReadEntireLog bool   `json:"ReadEntireLog"` // Read the entire log or only new
	LogPollRate   int    `json:"LogPollRate"`   // How often to read the log if it reaches EOF in seconds
}

func init() {
	readConfig()
	log.Printf("Configuration loaded:\n %+v\n", configuration)
}

func readConfig() error {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Println(err)
	}
	if _, err := os.Stat(dir + "/" + configPath); os.IsNotExist(err) {
		// path/to/whatever does not exist
		dir, _ = os.Getwd()
	}
	if _, err := os.Stat(dir + "/" + configPath); os.IsNotExist(err) {
		// path/to/whatever does not exist
		dir = failPath
	}
	if _, err := os.Stat(dir + "/" + configPath); os.IsNotExist(err) {
		// path/to/whatever does not exist
		log.Fatal(err)
	}
	file, err := os.OpenFile(dir+"/"+configPath, os.O_RDONLY, 0444)
	defer file.Close()
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&configuration)
	if err != nil {
		return err
	}
	return nil
}

func saveConfig() error {
	marshalledConfig, _ := json.MarshalIndent(configuration, "", "\t")
	err := ioutil.WriteFile(configPath, marshalledConfig, 0644)
	if err != nil {
		return err
	}
	log.Printf("Config Saved to %s\n", configPath)
	return nil
}
