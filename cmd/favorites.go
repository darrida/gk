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
	"github.com/tobischo/gokeepasslib/v3"
)

func init() {
	rootCmd.AddCommand(favoritesCmd)
	// main favorites command
	favoritesCmd.AddCommand(favoritesList)
	favoritesCmd.Flags().BoolP("password", "p", false, "Print password to stdout")
	favoritesCmd.Flags().BoolP("copy", "c", false, "Copy password to clipboard")
	favoritesCmd.Flags().BoolP("test", "t", false, "Run CLI command in test mode (no user prompts)")
	// Add alias for favorites command
	rootCmd.AddCommand(favCmd)
	favCmd.AddCommand(favoritesList)
	favCmd.Flags().BoolP("password", "p", false, "Print password to stdout")
	favCmd.Flags().BoolP("copy", "c", false, "Copy password to clipboard")
	favCmd.Flags().BoolP("test", "t", false, "Run CLI command in test mode (no user prompts)")
	// Add search flags
	// favoritesList.Flags().StringP("favorite", "f", "", "Select favorite using index")
	// favoritesList.Flags().StringP("search", "s", "", "Search favorites by title, username, URL")
	// favoritesList.Flags().BoolP("exact", "e", false, "Exact match only (no fuzzy search)")
	favoritesList.Flags().BoolP("detail", "d", false, "Show details of entries")
	favoritesList.Flags().BoolP("test", "t", false, "Run CLI command in test mode (no user prompts)")
}

var favCmd = &cobra.Command{
	Use:   "fav [INDEX]",
	Short: "Alias of `favorites`",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		showPassword, _ := cmd.Flags().GetBool("password")
		copyToClipboard, _ := cmd.Flags().GetBool("copy")
		test, _ := cmd.Flags().GetBool("test")

		if len(args) == 0 {
			fmt.Println("Index argument required. Use `gokp favorites list` to see all favorites.")
			os.Exit(0)
		}

		index, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Printf("Invalid index: %s\n", args[0])
			return
		}
		showFavoriteEntry(index, showPassword, copyToClipboard, test)
	},
}

var favoritesCmd = &cobra.Command{
	Use:   "favorites [INDEX]",
	Short: "Use and manage favorites entries",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		showPassword, _ := cmd.Flags().GetBool("password")
		copyToClipboard, _ := cmd.Flags().GetBool("copy")
		test, _ := cmd.Flags().GetBool("test")

		if len(args) == 0 {
			fmt.Println("Index argument required. Use `gokp favorites list` to see all favorites.")
			os.Exit(0)
		}

		index, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Printf("Invalid index: %s\n", args[0])
			return
		}
		showFavoriteEntry(index, showPassword, copyToClipboard, test)
	},
}

var favoritesList = &cobra.Command{
	Use:   "list",
	Short: "List favorites from external Keepass databases",
	// Args:  cobra.ExactArgs(1),
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

		var favGroupIndex int
		for i := range db.Content.Root.Groups {
			if db.Content.Root.Groups[i].Name == "favorites" {
				favGroupIndex = i
				break
			}
		}

		for _, entry := range db.Content.Root.Groups[favGroupIndex].Entries {
			index := entry.GetContent("Favorite Index")
			printFavoritesResult(index, entry, "favorites")
		}
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

	fmt.Printf("\n%s------ Entry -------%s\n", ColorBoldCyan, ColorReset)
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
		fmt.Printf("%s----- Password -----%s\n", ColorBoldCyan, ColorReset)

		if showPassword {
			fmt.Println(entry.GetPassword())
		}
	}
	fmt.Printf("%s--------------------%s\n", ColorBoldCyan, ColorReset)

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

		config := readConfig()
		fmt.Printf("\nPassword for favorite #%d copied to clipboard\n", index)
		showCountdownBarWithSignalHandling(config.ClipboardTimeout, originalClipboard, entry.GetPassword())
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

	fmt.Printf("\nClipboard will clear in %d seconds (Press %sCtrl+C%s to exit early and clear now):\n", seconds, ColorBoldRed, ColorReset)

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
				fmt.Printf("\r[%s] %2ds", bar, i)
			}
		}
	}()

	// Wait for either completion or signal
	select {
	case <-sigChan:
		fmt.Printf("\n\n🛑 Interrupted! Clearing clipboard now...\n\n")
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
			fmt.Printf("✅ Clipboard restored to previous content\n")
		} else {
			clipboard.WriteAll("")
			fmt.Printf("✅ Clipboard cleared\n")
		}
	} else {
		fmt.Printf("Clipboard was changed by user - not modifying\n")
	}
}

func printFavoritesResult(count string, entry gokeepasslib.Entry, databaseName string) {
	title := entry.GetTitle()
	username := getEntryValue(entry, "UserName")
	url := getEntryValue(entry, "URL")

	fmt.Printf("\n-------------------------------------------------------\n")
	if count != "" {
		fmt.Printf(
			"[%s%s%s] | Title: %s%s%s | Username: %s%s%s\n",
			ColorBoldGreen, count, ColorReset, ColorBoldCyan, title, ColorReset, ColorBoldCyan, username, ColorReset,
		)
		fmt.Printf("-------------------------------------------------------\n")
	}
	fmt.Printf("Database:  %s\n", databaseName)
	if url != "" {
		fmt.Printf("URL:       %s\n", url)
	}

	var firstPass bool = true
	for _, value := range entry.Values {
		if value.Key == "Database Source" || value.Key == "Database path" {
			if firstPass {
				fmt.Println("Custom Attributes:")
				firstPass = false
			}
			fmt.Printf("- %s: %s\n", value.Key, value.Value.Content)
		}
	}
}
