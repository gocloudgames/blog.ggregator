package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

const configFileName = ".gatorconfig.json"

func ReadConfig() (Config, error) {
	cfg := Config{}

	cfgFile := getConfigFilePath()
	// Open the file
	file, err := os.Open(cfgFile)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return cfg, err
	}
	defer file.Close()

	// Read file content
	bytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return cfg, err
	}

	err = json.Unmarshal(bytes, &cfg)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return cfg, err
	}

	return cfg, nil
}

func (cfg Config) SetUser() error {
	cfgFile := getConfigFilePath()

	// Create or truncate the file
	file, err := os.Create(cfgFile)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return err
	}
	defer file.Close()

	// Encode struct to JSON and write to file
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // for pretty output (optional)
	if err := encoder.Encode(cfg); err != nil {
		fmt.Println("Error encoding JSON:", err)
	} else {
		fmt.Println("User written to user.json")
	}
	return nil
}

func getConfigFilePath() string {
	// if use $HOME directory, use functiob os.UserHomeDir()
	configDir, _ := os.UserHomeDir()
	return filepath.Join(configDir, configFileName)
}
