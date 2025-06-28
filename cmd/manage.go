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
	var TestMode bool
	var SetupEntry bool
	openCmd.PersistentFlags().BoolVarP(&TestMode, "test", "t", false, "Run CLI command in test mode")
	openCmd.PersistentFlags().BoolVarP(&SetupEntry, "setup", "s", false, "Setup a new keepass database entry")
}

var manageDbsCmd = &cobra.Command{
	Use:   "manage",
	Short: "Manage Entries to External Keepass DBs",
}

var addDbCmd = &cobra.Command{
	Use:   "add [NAME]",
	Short: "Add new Keepass Database",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		entry_name := args[0]
		kdbx_path := "test.kdbx"
		kdbx_key := "" //"test.key"

		_, _, gokpKDBX := pathSelection(false)

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
		file.Close()

		db.UnlockProtectedEntries()

		index := FindRootGroupIndexByName(db.Content.Root.Groups, "databases")
		if index == nil || len(db.Content.Root.Groups) < *index {
			fmt.Println("ERROR: `databases` group not found in gokp.kdbx. Try running `gokp setup init`")
			os.Exit(1)
		}
		groupIndex := *index
		databasesGroup := db.Content.Root.Groups[groupIndex]

		fmt.Println(databasesGroup.Name)
		for _, entry := range databasesGroup.Entries {
			if entry_name == entry.GetTitle() {
				fmt.Println("\nERROR: Database entry by that name already exists.")
				os.Exit(0)
			}
		}

		newEntry := gokeepasslib.NewEntry()
		newEntry.Values = append(newEntry.Values, mkValue("Title", entry_name))
		newEntry.Values = append(newEntry.Values, mkValue("UserName", kdbx_path))
		newEntry.Values = append(newEntry.Values, mkProtectedValue("Password", "123456"))
		newEntry.Values = append(newEntry.Values, mkValue("URL", kdbx_key))
		fmt.Println(newEntry.GetTitle())

		// Add the new entry to the target group
		databasesGroup.Entries = append(databasesGroup.Entries, newEntry)
		fmt.Printf("Entry added to group '%s'\n", databasesGroup.Name)

		db.Content.Root.Groups[groupIndex] = databasesGroup

		db.LockProtectedEntries()

		// SAVE CHANGES
		writeFile, err := os.Create(gokpKDBX)
		if err != nil {
			panic(err)
		}
		defer writeFile.Close()

		keepassEncoder := gokeepasslib.NewEncoder(writeFile)
		if err := keepassEncoder.Encode(db); err != nil {
			panic(err)
		}
	},
}

var openCmd = &cobra.Command{
	Use:   "open [NAME]",
	Short: "Open existing database",
	// Long:  "Open existing database",
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			fmt.Errorf("Too many arguments passed. Cancelled.")
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
		fmt.Println(databases.Name)
		for _, entry := range databases.Entries {
			fmt.Print(entry.GetTitle())
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
