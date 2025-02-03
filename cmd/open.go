package cmd

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tobischo/gokeepasslib/v3"
	"golang.org/x/term"
)

// "fmt"
// "log"
// "os"
// "path/filepath"
// "syscall"

// w "github.com/tobischo/gokeepasslib/v3/wrappers"
// "golang.org/x/term"
var openCmd = &cobra.Command{
	Use:   "open [NAME]",
	Short: "Open existing database",
	// Long:  "Open existing database",
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Errorf("Too many arguments passed. Cancelled.")
		}
		name := strings.Join(args, "")
		println(name)

		test, _ := cmd.Flags().GetBool("test")
		setup, _ := cmd.Flags().GetBool("setup")
		println(setup)

		_, _, gokpKDBX := pathSelection(test)

		fmt.Print("Enter admin password: ")
		password, _ := term.ReadPassword(int(syscall.Stdin))
		if string(password) == "" {
			fmt.Println("\nPassword is required")
			os.Exit(1)
		}
		fmt.Println()
		passwordStr := string(password)

		file, _ := os.Open(gokpKDBX)
		db := gokeepasslib.NewDatabase()
		db.Credentials = gokeepasslib.NewPasswordCredentials(passwordStr)
		_ = gokeepasslib.NewDecoder(file).Decode(db)

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

func init() {
	rootCmd.AddCommand(openCmd)
	var TestMode bool
	var SetupEntry bool
	openCmd.PersistentFlags().BoolVarP(&TestMode, "test", "t", false, "Run CLI command in test mode")
	openCmd.PersistentFlags().BoolVarP(&SetupEntry, "setup", "s", false, "Setup a new keepass database entry")
}

func setupEntry(name string) {

}
