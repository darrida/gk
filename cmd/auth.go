package cmd

import (
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
	"golang.org/x/term"
)

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(loginCmd)
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage gokp db password with OS keystore",
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove gokp password from OS keystore",
	Run: func(cmd *cobra.Command, args []string) {
		delete_password("gokp", "local")
		println("gokp password cleared.")
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Save gokp password to OS keystore",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("Enter admin password: ")
		password, _ := term.ReadPassword(int(syscall.Stdin))
		if string(password) == "" {
			fmt.Println("\nPassword is required")
			os.Exit(1)
		}
		fmt.Println()
		passwordStr := string(password)

		save_password("gokp", "local", passwordStr)
		println("Saved gokp password to keystore")
	},
}

func save_password(service string, user string, password string) {
	err := keyring.Set(service, user, password)
	if err != nil {
		log.Fatal(err)
	}
}

func get_password(service string, user string) (string, error) {
	secret, err := keyring.Get(service, user)
	return secret, err
}

func delete_password(service string, user string) {
	err := keyring.Delete(service, user)
	if err != nil {
		log.Fatal((err))
	}
}
