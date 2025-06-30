package cmd

// import (
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"os"
// )

// type ConfigTemp struct {
// 	ClipboardTimeout int    `json:"clipboardTimeout"` // Fix: should be int, not string
// 	DefaultDatabase  string `json:"defaultDatabase"`
// 	AutoClear        bool   `json:"autoClear"`
// }

// // Read config from JSON file
// func readConfigTemp() *Config {
// 	configFile := "config.json"
// 	config := &Config{
// 		// Set defaults
// 		ClipboardTimeout: 60,
// 	}

// 	data, err := os.ReadFile(configFile)
// 	if err != nil {
// 		// If file doesn't exist, create it with defaults
// 		if os.IsNotExist(err) {
// 			fmt.Println("Config file not found, creating with defaults...")
// 			saveConfigTemp(config)
// 			return config
// 		}
// 		log.Fatalf("Error reading config file: %v", err)
// 	}

// 	err = json.Unmarshal(data, config)
// 	if err != nil {
// 		log.Fatalf("Error unmarshalling JSON: %v", err)
// 	}

// 	return config
// }

// // Save config struct to JSON file
// func saveConfigTemp(config *Config) error {
// 	configFile := "config.json"

// 	// Marshal struct to JSON with indentation for readability
// 	data, err := json.MarshalIndent(config, "", "  ")
// 	if err != nil {
// 		return fmt.Errorf("error marshalling config to JSON: %v", err)
// 	}

// 	// Write to file with proper permissions
// 	err = os.WriteFile(configFile, data, 0644)
// 	if err != nil {
// 		return fmt.Errorf("error writing config file: %v", err)
// 	}

// 	fmt.Printf("Config saved to %s\n", configFile)
// 	return nil
// }

// // Update specific config values
// func updateConfigTemp(clipboardTimeout int) error {
// 	config := readConfigTemp()

// 	// Update values
// 	if clipboardTimeout > 0 {
// 		config.ClipboardTimeout = clipboardTimeout
// 	}

// 	return saveConfigTemp(config)
// }

// // Get config with fallback to defaults
// func getConfigTemp() *Config {
// 	config := readConfigTemp()

// 	// Validate and set defaults if needed
// 	if config.ClipboardTimeout <= 0 {
// 		config.ClipboardTimeout = 60
// 	}

// 	return config
// }

// // Example usage functions
// func createDefaultConfigTemp() {
// 	config := &Config{
// 		ClipboardTimeout: 60,
// 	}

// 	err := saveConfigTemp(config)
// 	if err != nil {
// 		log.Fatalf("Failed to create default config: %v", err)
// 	}
// }

// func printConfigTemp() {
// 	config := readConfigTemp()
// 	fmt.Printf("Current Configuration:\n")
// 	fmt.Printf("  Clipboard Timeout: %d seconds\n", config.ClipboardTimeout)
// }
