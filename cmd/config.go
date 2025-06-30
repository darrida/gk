package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.Flags().StringP("clipboard-timeout", "c", "", "Set clipboard timeout in seconds (default: 30)")
	configCmd.Flags().BoolP("test", "t", false, "Run CLI command in test mode")
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage gokp configuration",
	Long: `Manage gokp configuration settings.
This command allows you to view and modify the gokp configuration file.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printConfig()
			return
		}

		subCommand := args[0]
		config := readConfig()

		switch subCommand {
		case "read":
			fmt.Printf("Current Configuration:\n")
			fmt.Printf("- Clipboard Timeout: %d seconds\n", config.ClipboardTimeout)
		case "update":
			timeoutStr, _ := cmd.Flags().GetString("clipboard-timeout")
			if timeoutStr == "" {
				fmt.Println("ERROR: Please provide a clipboard timeout value using --clipboard-timeout flag.")
				os.Exit(0)
			}
			if timeoutStr != "" {
				timeoutInt, err := strconv.Atoi(timeoutStr)
				if err != nil || timeoutInt <= 0 {
					log.Fatal("Invalid clipboard timeout value. Must be a positive integer.")
				}
				config.ClipboardTimeout = timeoutInt
			}
			err := saveConfig(config)
			if err != nil {
				log.Fatalf("Error saving config: %v", err)
			}
			fmt.Println("Configuration updated successfully.")
		default:
			log.Fatalf("Unknown command: %s", args[0])
		}
	},
}

type Config struct {
	ClipboardTimeout int `json:"clipboard-timeout"`
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
