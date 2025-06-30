package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	ClipboardTimeout int `json:"clipboard-timeout`
}

func readConfig() *Config {

	gokpFolder, _, _ := pathSelection(false)
	configPath := filepath.Join(gokpFolder, "config.json")

	config := &Config{
		ClipboardTimeout: 30, // Default value
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Config file not found, creating with defaults...")
			config = createDefaultConfig()
			return config

		}
		log.Fatalf("Error reading config file: %v", err)
	}

	err = json.Unmarshal(data, config)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
	}

	return config
}

func saveConfig(config *Config) error {
	gokpFolder, _, _ := pathSelection(false)
	configPath := filepath.Join(gokpFolder, "config.json")

	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return fmt.Errorf("error marshalling config to JSON: %v", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	fmt.Printf("Config saved to %s\n", configPath)
	return nil
}

func updateConfig(clipboardTimeout int) error {
	config := readConfig()
	if clipboardTimeout > 0 {
		config.ClipboardTimeout = clipboardTimeout
	}
	return saveConfig(config)
}

func createDefaultConfig() *Config {
	config := &Config{
		ClipboardTimeout: 30,
	}

	error := saveConfig(config)
	if error != nil {
		log.Fatalf("Failed to create default config: %v", error)
	}

	return config
}

func printConfig() {
	config := readConfig()
	fmt.Printf("Current Configuration:\n")
	fmt.Printf("- Clipboard TImeout: %d seconds\n", config.ClipboardTimeout)
}
