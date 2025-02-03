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

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Initial setup of gokp app database",
	Run: func(cmd *cobra.Command, args []string) {
		test, _ := cmd.Flags().GetBool("test")

		gokpFolder, _, gokpKDBX := pathSelection(test)
		// println(gokpFolder)
		// println(gokpExecutable)
		// println(gokpKDBX)

		if _, err := os.Stat(gokpFolder); os.IsNotExist(err) {
			println("Creating .gokeepass folder in home directory")
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

		println(passwordStr)
		createDB(gokpKDBX, passwordStr)
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
	var TestMode bool
	setupCmd.PersistentFlags().BoolVarP(&TestMode, "test", "t", false, "Run CLI command in test mode")
}

func pathSelection(test bool) (string, string, string) {
	homeDir, _ := os.UserHomeDir()

	var gokpFolder string
	if test == false {
		gokpFolder = filepath.Join(homeDir, ".gokeepass")
	} else {
		gokpFolder = filepath.Join(homeDir, "test", ".gokeepass")
	}

	gokpExecutable := filepath.Join(gokpFolder, "keepass.exe")
	gokpKDBX := filepath.Join(gokpFolder, "gokeepass.kdbx")
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
	file, _ := os.Create(dbPath)
	defer file.Close()

	//
	//
	//
	// create root group
	rootGroup := gokeepasslib.NewGroup()
	rootGroup.Name = "root group"

	// entry := gokeepasslib.NewEntry()
	// entry.Values = append(entry.Values, mkValue("Title", "My GMail password"))
	// entry.Values = append(entry.Values, mkValue("UserName", "example@gmail.com"))
	// entry.Values = append(entry.Values, mkProtectedValue("Password", "hunter2"))

	// rootGroup.Entries = append(rootGroup.Entries, entry)

	// // demonstrate creating sub group (we'll leave it empty because we're lazy)
	// subGroup := gokeepasslib.NewGroup()
	// subGroup.Name = "sub group"

	// subEntry := gokeepasslib.NewEntry()
	// subEntry.Values = append(subEntry.Values, mkValue("Title", "Another password"))
	// subEntry.Values = append(subEntry.Values, mkValue("UserName", "johndough"))
	// subEntry.Values = append(subEntry.Values, mkProtectedValue("Password", "123456"))

	// subGroup.Entries = append(subGroup.Entries, subEntry)

	// rootGroup.Groups = append(rootGroup.Groups, subGroup)
	//
	//
	//
	//

	db := &gokeepasslib.Database{
		Header:      gokeepasslib.NewHeader(),
		Credentials: gokeepasslib.NewPasswordCredentials(password),
		Content: &gokeepasslib.DBContent{
			Meta: gokeepasslib.NewMetaData(),
			Root: &gokeepasslib.RootData{
				Groups: []gokeepasslib.Group{rootGroup},
			},
		},
	}
	// db.LockProtectedEntries()
	keepassEncoder := gokeepasslib.NewEncoder(file)
	if err := keepassEncoder.Encode(db); err != nil {
		panic(err)
	}
	// db := gokeepasslib.NewDatabase()
	// db.Credentials = gokeepasslib.NewPasswordCredentials(password)
	// _ = gokeepasslib.NewDecoder(file).Decode(db)
	println("\nDONE: gokp app database created.\nSetup keepass databases by using:\n- 'gokp open <NEW_NAME> -s'")

	// db.UnlockProtectedEntries()

	// // Note: This is a simplified example and the groups and entries will depend on the specific file.
	// // bound checking for the slices is recommended to avoid panics.
	// entry := db.Content.Root.Groups[0].Groups[0].Entries[0]
	// fmt.Println(entry.GetTitle())
	// fmt.Println(entry.GetPassword())
}
