package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "root-help",
	Short: "Manage multiple keepasses used on a daily basis",
	// Run: func(cmd *cobra.Command, args []string) {
	// 	println("To get started, use the help command")
	// },
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
