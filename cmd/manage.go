package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tobischo/gokeepasslib/v3"
	"golang.org/x/term"
)

func init() {
	rootCmd.AddCommand(manageDbsCmd)
	manageDbsCmd.AddCommand(openCmd)
	manageDbsCmd.AddCommand(addDbCmd)
	manageDbsCmd.AddCommand(listDbsCmd)
	var TestMode bool
	var SetupEntry bool
	openCmd.PersistentFlags().BoolVarP(&TestMode, "test", "t", false, "Run CLI command in test mode")
	openCmd.PersistentFlags().BoolVarP(&SetupEntry, "setup", "s", false, "Setup a new keepass database entry")

	// Add flags for addDbCmd
	addDbCmd.Flags().StringP("path", "p", "", "Path to the KeePass database file (required)")
	addDbCmd.Flags().StringP("password", "w", "", "Password for the database (required)")
	addDbCmd.Flags().StringP("key", "k", "", "Path to the key file (optional)")
	addDbCmd.MarkFlagRequired("path")
}

var manageDbsCmd = &cobra.Command{
	Use:   "manage",
	Short: "Manage Entries to External Keepass DBs",
}

var addDbCmd = &cobra.Command{
	Use:   "add [NAME] --path PATH --password PASSWORD [--key KEYFILE]",
	Short: "Add new Keepass Database",
	Long: `Add a new KeePass database entry to the GoKP database.

Examples:
  gokp manage add mydb --path /path/to/database.kdbx
  gokp manage add mydb --path /path/to/database.kdbx --key /path/to/keyfile.key`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		entry_name := args[0]

		// Get path and key from command-line flags
		kdbx_path, _ := cmd.Flags().GetString("path")
		kdbx_key, _ := cmd.Flags().GetString("key")
		kdbx_password, _ := cmd.Flags().GetString("password")

		// Validate that the database file exists
		if _, err := os.Stat(kdbx_path); os.IsNotExist(err) {
			log.Fatalf("Database file does not exist: %s", kdbx_path)
		}

		// Validate key file if provided
		if kdbx_key != "" {
			if _, err := os.Stat(kdbx_key); os.IsNotExist(err) {
				log.Fatalf("Key file does not exist: %s", kdbx_key)
			}
		}

		_, _, gokpKDBX := pathSelection(false)

		secret, err := getGoKPPassword()
		if err != nil {
			log.Fatalf("Failed to get GoKP password: %v", err)
		}

		db, err := openKeepassDB(gokpKDBX, secret)
		if err != nil {
			log.Fatalf("Failed to open Keepass database: %v", err)
		}

		existingEntry := readEntryFromGroup(db, "databases", entry_name)
		if existingEntry != nil {
			fmt.Printf("\nERROR: Database entry by the name '%s' already exists.\n", entry_name)
			os.Exit(0)
		}

		addGoKPEntryToGroup(db, "databases", entry_name, kdbx_password, kdbx_path, kdbx_key)

		err = saveKeepassDB(db, gokpKDBX)
		if err != nil {
			log.Fatalf("Failed to save Keepass database: %v", err)
		} else {
			fmt.Printf("\nSuccessfully added new database entry '%s' to the GoKP database.\n", entry_name)
			fmt.Printf("Database path: %s\n", kdbx_path)
			if kdbx_key != "" {
				fmt.Printf("Key file: %s\n", kdbx_key)
			}
		}
	},
}

var listDbsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all databases in the GoKP database",
	Run: func(cmd *cobra.Command, args []string) {
		test, _ := cmd.Flags().GetBool("test")
		_, _, gokpKDBX := pathSelection(test)

		secret, err := getGoKPPassword()
		if err != nil {
			log.Fatalf("Failed to get GoKP password: %v", err)
		}

		db, err := openKeepassDB(gokpKDBX, secret)
		if err != nil {
			log.Fatalf("Failed to open Keepass database: %v", err)
		}

		databases := FindRootGroupByName(db.Content.Root.Groups, "databases")
		fmt.Printf("\nFound group: %s\n", databases.Name)
		fmt.Println("Databases:")
		for _, entry := range databases.Entries {
			fmt.Printf("- %s\n    Path: %s\n", entry.GetTitle(), entry.GetContent("Database Path"))
			if entry.GetContent("Key File Path") != "" {
				fmt.Printf("    Key:  %s\n", entry.GetContent("Key File Path"))
			}
		}
	},
}

var openCmd = &cobra.Command{
	Use:   "open [NAME]",
	Short: "Open existing database",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			log.Fatal("Too many arguments passed. Cancelled.")
		}
		name := strings.Join(args, "")
		println(name)

		test, _ := cmd.Flags().GetBool("test")
		setup, _ := cmd.Flags().GetBool("setup")
		println(setup)

		_, _, gokpKDBX := pathSelection(test)

		secret, err := get_password("gokp", "local")
		if err != nil {
			fmt.Print("Enter admin password: ")
			password, _ := term.ReadPassword(int(syscall.Stdin))
			if string(password) == "" {
				fmt.Println("\nPassword is required")
				os.Exit(1)
			}
			fmt.Println()
			secret = string(password)
		}

		file, err := os.Open(gokpKDBX)
		if err != nil {
			log.Fatal(err)
		}
		db := gokeepasslib.NewDatabase()
		db.Credentials = gokeepasslib.NewPasswordCredentials(secret)
		err = gokeepasslib.NewDecoder(file).Decode(db)
		if err != nil {
			println("\nWARNING: Unable to open gokeepass db. The password is likely incorrect.")
			os.Exit(1)
		}

		db.UnlockProtectedEntries()

		// entry := gokeepasslib.NewEntry()
		// entry.Values = append(entry.Values, mkValue("Title", name))
		// // entry.Values = append(entry.Values, mkValue("UserName", name))
		// // entry.Values = append(entry.Values, mkProtectedValue("Password", "hunter2"))
		// rootGroup.Entries = append(rootGroup.Entries, entry)
		// rootGroup.Group = append()

		// db.Content = &gokeepasslib.DBContent{
		// 	Meta: &gokeepasslib.MetaData{},
		// 	Root: &gokeepasslib.RootData{
		// 		Groups: []gokeepasslib.Group{rootGroup},
		// 	},
		// }
		databases := FindRootGroupByName(db.Content.Root.Groups, "databases")
		fmt.Printf("\nFound group: %s\n", databases.Name)
		fmt.Println("Databases:")
		for _, entry := range databases.Entries {
			fmt.Printf("\n--- %s ---\n", entry.GetTitle())

			// Display all attributes
			for _, value := range entry.Values {
				isProtected := value.Value.Protected.Bool
				if value.Key == "Password" || isProtected {
					fmt.Printf("%s: [PROTECTED]\n", value.Key)
				} else {
					fmt.Printf("%s: %s\n", value.Key, value.Value.Content)
				}
			}
		}

		// entry := db.Content.Root.Groups[0].Groups[0].Entries[0]
		// fmt.Println(entry.GetTitle())
		// fmt.Println(entry.GetPassword())
		// println(gokpFolder)
		// println(gokpExecutable)
		// println(gokpKDBX)

		// println(passwordStr)
		// createDB(gokpKDBX, passwordStr)
	},
}

func setupEntry(name string) {

}
