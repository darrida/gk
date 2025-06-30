package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
	"golang.org/x/term"
)

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.AddCommand(setupInitCmd)
	setupCmd.AddCommand(setupDeleteCmd)
	var TestMode bool
	setupInitCmd.PersistentFlags().BoolVarP(&TestMode, "test", "t", false, "Run CLI command in test mode")
	setupDeleteCmd.PersistentFlags().BoolVarP(&TestMode, "test", "t", false, "Run CLI command in test mode")
	setupDeleteCmd.Flags().BoolP("force", "f", false, "Force deletion without confirmation prompts")
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Manage gokp database",
}

var setupInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initial setup of gokp app database",
	Run: func(cmd *cobra.Command, args []string) {
		test, _ := cmd.Flags().GetBool("test")

		gokpFolder, _, gokpKDBX := pathSelection(test)
		// println(gokpFolder)
		// println(gokpExecutable)
		// println(gokpKDBX)

		if _, err := os.Stat(gokpFolder); os.IsNotExist(err) {
			println("Creating .gokp folder in home directory")
			os.Mkdir(gokpFolder, os.ModePerm)
		}

		if _, err := os.Stat(gokpKDBX); !os.IsNotExist(err) {
			fmt.Print("\nWARNING: If an app database already exists, this process will delete it and create a fresh one.\nProceed (yes/no)? ")

			var confirmation string
			fmt.Scanln(&confirmation)

			if confirmation != "yes" {
				print("END: gokp setup cancelled")
				os.Exit(0)
			}
		}

		if _, err := os.Stat(gokpKDBX); !os.IsNotExist(err) {
			err := os.Remove(gokpKDBX)
			if err != nil {
				log.Fatal(err)
			}
		}
		println("\nSTEP 1: Create gokp app database.")
		fmt.Print("Enter admin password: ")
		password, _ := term.ReadPassword(int(syscall.Stdin))
		if string(password) == "" {
			fmt.Println("\nPassword is required")
			os.Exit(1)
		}
		fmt.Println()
		passwordStr := string(password)

		fmt.Print("\nWould you like to save this password to the local OS key store (yes/no)? ")
		var confirmation string
		fmt.Scanln(&confirmation)
		if confirmation == "yes" {
			save_password("gokp", "local", passwordStr)
			println("Saved gokeepass password to keystore")
		}

		fmt.Printf("\nCreating default config.json in %s\n", gokpFolder)
		config := createDefaultConfig()
		if config == nil {
			configPath := filepath.Join(gokpFolder, "config.json")
			log.Fatalf("ERROR: Failed to create default config at %s\n", configPath)
		}

		createDB(gokpKDBX, passwordStr)
	},
}

var setupDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete the gokp app database",
	Long: `Delete the gokp app database file and optionally remove the password from keystore.

WARNING: This will permanently delete your gokp database and all stored database entries.
Make sure to backup any important data before proceeding.`,
	Run: func(cmd *cobra.Command, args []string) {
		test, _ := cmd.Flags().GetBool("test")
		force, _ := cmd.Flags().GetBool("force")

		gokpFolder, _, gokpKDBX := pathSelection(test)

		// Check if database exists
		if _, err := os.Stat(gokpKDBX); os.IsNotExist(err) {
			fmt.Println("No gokp database found to delete.")
			return
		}

		if !force {
			fmt.Printf("WARNING: This will permanently delete your gokp database at:\n%s\n", gokpKDBX)
			fmt.Print("\nAre you sure you want to proceed? (yes/no): ")

			var confirmation string
			fmt.Scanln(&confirmation)

			if confirmation != "yes" {
				fmt.Println("Deletion cancelled.")
				return
			}
		}

		// Delete the database file
		err := os.Remove(gokpKDBX)
		if err != nil {
			log.Fatalf("Failed to delete database file: %v", err)
		}

		fmt.Printf("Successfully deleted gokp database: %s\n", gokpKDBX)

		// Handle keystore password removal
		removePassword := force
		if !force {
			fmt.Print("\nWould you like to remove the stored password from keystore as well? (yes/no): ")
			var response string
			fmt.Scanln(&response)
			removePassword = (response == "yes")
		}

		if removePassword {
			delete_password("gokp", "local")
			fmt.Println("Password removed from keystore.")
		}

		// Handle config.json removal

		configPath := filepath.Join(gokpFolder, "config.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			fmt.Println("No config.json found to delete.")
		} else {
			fmt.Printf("\nWould you like to remove config.json from %s? (yes/no): ", configPath)
			var response string
			fmt.Scanln(&response)
			_, err := os.ReadFile(gokpFolder)
			if err != nil {
				fmt.Printf("Warning: Failed to remove %s: %v\n", configPath, err)
			} else {
				fmt.Printf("Removed: %s\n", configPath)
			}
		}

		// Handle folder removal
		entries, err := os.ReadDir(gokpFolder)
		if err == nil && len(entries) == 0 {
			removeFolder := force
			if !force {
				fmt.Print("\nThe .gokp folder is now empty. Remove it as well? (yes/no): ")
				var response string
				fmt.Scanln(&response)
				removeFolder = (response == "yes")
			}

			if removeFolder {
				err := os.Remove(gokpFolder)
				if err != nil {
					fmt.Printf("Warning: Failed to remove folder %s: %v\n", gokpFolder, err)
				} else {
					fmt.Printf("Removed folder: %s\n", gokpFolder)
				}
			}
		}

		fmt.Println("\nGokp database deletion completed.")
	},
}

func pathSelection(test bool) (string, string, string) {
	homeDir, _ := os.UserHomeDir()

	var gokpFolder string
	if !test {
		gokpFolder = filepath.Join(homeDir, ".gokp")
	} else {
		gokpFolder = filepath.Join(homeDir, "test", ".gokp")
	}

	gokpExecutable := filepath.Join(gokpFolder, "keepass.exe")
	gokpKDBX := filepath.Join(gokpFolder, "gokp.kdbx")
	return gokpFolder, gokpExecutable, gokpKDBX
}

func mkValue(key string, value string) gokeepasslib.ValueData {
	return gokeepasslib.ValueData{Key: key, Value: gokeepasslib.V{Content: value}}
}

func mkProtectedValue(key string, value string) gokeepasslib.ValueData {
	return gokeepasslib.ValueData{
		Key:   key,
		Value: gokeepasslib.V{Content: value, Protected: w.NewBoolWrapper(true)},
	}
}

func createDB(dbPath string, password string) {
	// file, _ := os.Create(dbPath)
	// defer file.Close()

	dbsGroup := gokeepasslib.NewGroup()
	dbsGroup.Name = "databases"

	favGroup := gokeepasslib.NewGroup()
	favGroup.Name = "favorites"

	db := &gokeepasslib.Database{
		Header:      gokeepasslib.NewHeader(),
		Credentials: gokeepasslib.NewPasswordCredentials(password),
		Content: &gokeepasslib.DBContent{
			Meta: gokeepasslib.NewMetaData(),
			Root: &gokeepasslib.RootData{
				Groups: []gokeepasslib.Group{dbsGroup, favGroup},
			},
		},
	}

	err := FindRootGroupByName(db.Content.Root.Groups, dbsGroup.Name)
	if err == nil {
		log.Fatalf("ERROR: Failed to find root group by name: %s", dbsGroup.Name)
	}
	err = FindRootGroupByName(db.Content.Root.Groups, favGroup.Name)
	if err == nil {
		log.Fatalf("ERROR: Failed to find root group by name: %s", favGroup.Name)
	}

	saveKeepassDB(db, dbPath)
	println("\nDONE: gokp app database created.\n\nFor information on setting up external keypass entrys: `gokp manage --help`")
}

func FindRootGroupByName(groups []gokeepasslib.Group, name string) *gokeepasslib.Group {
	for _, group := range groups {
		if group.Name == name {
			return &group
		}
	}
	return nil
}

func FindRootGroupIndexByName(groups []gokeepasslib.Group, name string) *int {
	for index, group := range groups {
		if group.Name == name {
			return &index
		}
	}
	return nil
}
