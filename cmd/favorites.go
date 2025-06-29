package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(favoritesCmd)
	favoritesCmd.AddCommand(favoritesList)
	// favoritesCmd.AddCommand(favoritesSelect)

	// Add alias for favorites command
	rootCmd.AddCommand(favCmd)
	favCmd.AddCommand(favoritesList)
	// favCmd.AddCommand(favoritesSelect)

	// Add main flags
	favCmd.Flags().BoolP("password", "p", false, "Print password to stdout")
	favCmd.Flags().BoolP("copy", "c", false, "Copy password to clipboard")
	favCmd.Flags().BoolP("test", "t", false, "Run CLI command in test mode (no user prompts)")
	// Add search flags
	favoritesList.Flags().StringP("favorite", "f", "", "Select favorite using index")
	favoritesList.Flags().StringP("search", "s", "", "Search favorites by title, username, URL")
	favoritesList.Flags().BoolP("exact", "e", false, "Exact match only (no fuzzy search)")
	favoritesList.Flags().StringP("database", "d", "", "Search for favorites only from a specific external database")
}

var favCmd = &cobra.Command{
	Use:   "fav [INDEX]",
	Short: "Alias of `favorites`",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		showPassword, _ := cmd.Flags().GetBool("password")
		copyToClipboard, _ := cmd.Flags().GetBool("copy")
		test, _ := cmd.Flags().GetBool("test")

		if len(args) > 0 {
			index, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Printf("Invalid index: %s\n", args[0])
				return
			}
			showFavoriteEntry(index, showPassword, copyToClipboard, test)
		} else {
			fmt.Println("No index provided, listing all favorites")
			// Call the list function here
		}
	},
}

var favoritesCmd = &cobra.Command{
	Use:   "favorites [INDEX]",
	Short: "Manage favorites from external Keepass databases",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		showPassword, _ := cmd.Flags().GetBool("password")
		copyToClipboard, _ := cmd.Flags().GetBool("copy")
		test, _ := cmd.Flags().GetBool("test")

		if len(args) > 0 {
			index, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Printf("Invalid index: %s\n", args[0])
				return
			}
			showFavoriteEntry(index, showPassword, copyToClipboard, test)
		} else {
			fmt.Println("No index provided, listing all favorites")
			// Call the list function here
		}
	},
}

var favoritesList = &cobra.Command{
	Use:   "list",
	Short: "List favorites from external Keepass databases",
	// Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("This will list favorites")
	},
}

// var favoritesSelect = &cobra.Command{
// 	Use:   "fav",
// 	Short: "List favorites from external Keepass databases",
// 	Args:  cobra.ExactArgs(1),
// 	Run: func(cmd *cobra.Command, args []string) {

// 	},
// }

func showFavoriteEntry(index int, showPassword, copyToClipboard bool, test bool) {
	_, _, gokpKDBX := pathSelection(test)

	secret, err := getGoKPPassword()
	if err != nil {
		log.Fatalf("Failed to get GoKP password: %v", err)
	}

	db, err := openKeepassDB(gokpKDBX, secret)
	if err != nil {
		log.Fatalf("Failed to open Keepass database: %v", err)
	}

	// Get the favorite entry password by index
	entry := readFavoritesEntry(db, index)
	if entry == nil {
		fmt.Printf("No favorite found at index %d\n", index)
		return
	}

	fmt.Printf("\n------ Entry -------\n")
	fmt.Printf("Favorite: %d\n", index)
	fmt.Printf("Title:    %s\n", entry.GetTitle())
	if entry.GetContent("UserName") != "" {
		fmt.Printf("Username: %s\n", entry.GetContent("UserName"))
	}
	if entry.GetContent("URL") != "" {
		fmt.Printf("URL:      %s\n", entry.GetContent("URL"))
	}
	if showPassword {
		fmt.Printf("")
		fmt.Printf("----- Password -----\n")

		if showPassword {
			fmt.Println(entry.GetPassword())
		}
	}
	fmt.Printf("--------------------\n")

	if !showPassword && !copyToClipboard {
		fmt.Printf("Favorite #%d found. Use -p to show password or -c to copy to clipboard\n", index)
	}

	if copyToClipboard {
		// Store current clipboard content
		originalClipboard, _ := clipboard.ReadAll()

		// Copy password to clipboard
		err := clipboard.WriteAll(entry.GetPassword())
		if err != nil {
			fmt.Printf("Failed to copy password to clipboard: %v\n", err)
			return
		}

		fmt.Printf("\nPassword for favorite #%d copied to clipboard\n", index)
		showCountdownBarWithSignalHandling(60, originalClipboard, entry.GetPassword())
	}
}

// Enhanced countdown with signal handling
func showCountdownBarWithSignalHandling(seconds int, originalClipboard, password string) {
	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel to signal completion
	done := make(chan bool, 1)

	// Set up signal handling for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	fmt.Printf("\nClipboard will clear in %d seconds (Press Ctrl+C to exit early and clear now):\n", seconds)

	// Start countdown goroutine
	go func() {
		defer func() { done <- true }()

		barWidth := 50
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for i := seconds; i > 0; i-- {
			select {
			case <-ctx.Done():
				return // Context cancelled
			case <-ticker.C:
				// Calculate progress
				progress := float64(seconds-i) / float64(seconds)
				filled := int(progress * float64(barWidth))

				// Create progress bar
				bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

				// Print progress bar with timer
				fmt.Printf("\r[%s] %2ds remaining", bar, i)
			}
		}
	}()

	// Wait for either completion or signal
	select {
	case <-sigChan:
		fmt.Printf("\n\n🛑 Interrupted! Clearing clipboard now...\n")
		cancel() // Cancel the countdown
		clearClipboardSafely(originalClipboard, password)
		os.Exit(0)
	case <-done:
		// Countdown completed normally
		fmt.Printf("\r%s\r", strings.Repeat(" ", 60))
		clearClipboardSafely(originalClipboard, password)
	}
}

// Helper function to safely clear clipboard
func clearClipboardSafely(originalClipboard, password string) {
	currentClipboard, err := clipboard.ReadAll()
	if err == nil && currentClipboard == password {
		if originalClipboard != "" {
			clipboard.WriteAll(originalClipboard)
			fmt.Printf("✓ Clipboard restored to previous content\n")
		} else {
			clipboard.WriteAll("")
			fmt.Printf("✓ Clipboard cleared\n")
		}
	} else {
		fmt.Printf("✓ Clipboard was changed by user - not modifying\n")
	}
}

// // Alternative: Simple version that just waits and handles signals
// func showCountdownWithExit(seconds int, originalClipboard, password string) {
// 	// Set up signal handling
// 	sigChan := make(chan os.Signal, 1)
// 	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

// 	fmt.Printf("Password copied to clipboard. Will clear in %d seconds (Ctrl+C to exit and clear now)\n", seconds)

// 	// Create timer
// 	timer := time.NewTimer(time.Duration(seconds) * time.Second)

// 	select {
// 	case <-sigChan:
// 		fmt.Println("\nInterrupted! Clearing clipboard...")
// 		clearClipboardSafely(originalClipboard, password)
// 		os.Exit(0)
// 	case <-timer.C:
// 		clearClipboardSafely(originalClipboard, password)
// 	}
// }
