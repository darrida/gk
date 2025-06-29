package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/tobischo/gokeepasslib/v3"
)

func init() {
	rootCmd.AddCommand(searchCmd)

	// Add search flags
	searchCmd.Flags().BoolP("case-sensitive", "c", false, "Perform case-sensitive search")
	searchCmd.Flags().BoolP("exact", "e", false, "Exact match only (no fuzzy search)")
	searchCmd.Flags().StringP("group", "g", "", "Search only in specific group")
	searchCmd.Flags().StringP("database", "d", "", "Search only in specific external database")
}

var searchCmd = &cobra.Command{
	Use:   "search [QUERY]",
	Short: "Search for entries in the external Keepass databases",
	Long: `Search for entries in the external Keepass databases referenced in the GoKP database.

The search will load all external databases referenced in the GoKP "databases" group
and search their entries for matching titles, usernames, URLs, notes, and custom fields.
By default, performs case-insensitive fuzzy search across all groups in all databases.

Examples:
  gokp search gmail                    # Fuzzy search for "gmail" across all external DBs
  gokp search -e "My Email"            # Exact match only
  gokp search -c Gmail                 # Case-sensitive search
  gokp search -g Personal mypassword   # Search only in "Personal" group`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		// Get search options
		caseSensitive, _ := cmd.Flags().GetBool("case-sensitive")
		exactMatch, _ := cmd.Flags().GetBool("exact")
		targetGroup, _ := cmd.Flags().GetString("group")
		targetDatabase, _ := cmd.Flags().GetString("database")

		_, _, gokpKDBX := pathSelection(false)

		secret, err := getGoKPPassword()
		if err != nil {
			log.Fatalf("Failed to get GoKP password: %v", err)
		}

		gokpDB, err := openKeepassDB(gokpKDBX, secret)
		if err != nil {
			log.Fatalf("Failed to open GoKP database: %v", err)
		}

		// Get all external database entries from the "databases" group
		databasesGroup := FindRootGroupByName(gokpDB.Content.Root.Groups, "databases")
		if databasesGroup == nil {
			fmt.Println("No databases group found in GoKP database.")
			return
		}

		if len(databasesGroup.Entries) == 0 {
			fmt.Println("No external databases configured. Use 'gokp manage add' to add databases first.")
			return
		}

		// Search across all external databases
		var allResults []SearchResult
		totalDBsSearched := 0

		for _, dbEntry := range databasesGroup.Entries {
			dbName := dbEntry.GetTitle()

			if targetDatabase != "" && dbName != targetDatabase {
				continue
			}

			dbPath := getEntryAttribute(&dbEntry, "Database Path")
			dbPassword := dbEntry.GetPassword()
			keyFilePath := getEntryAttribute(&dbEntry, "Key File Path")

			if dbPath == "" {
				fmt.Printf("Warning: Database '%s' has no path configured, skipping.\n", dbName)
				continue
			}

			// Check if database file exists
			if _, err := os.Stat(dbPath); os.IsNotExist(err) {
				fmt.Printf("Warning: Database file '%s' not found for database '%s', skipping.\n", dbPath, dbName)
				continue
			}

			// Open external database
			externalDB, err := openExternalKeepassDB(dbPath, dbPassword, keyFilePath)
			if err != nil {
				fmt.Printf("Warning: Failed to open database '%s': %v, skipping.\n", dbName, err)
				continue
			}

			totalDBsSearched++

			// Search in this external database
			var results []gokeepasslib.Entry
			if targetGroup != "" {
				// Search in specific group
				group := FindRootGroupByName(externalDB.Content.Root.Groups, targetGroup)
				if group != nil {
					results = searchEntriesInGroup(group, query, caseSensitive, exactMatch)
				}
			} else {
				// Search in all groups
				results = fuzzySearchEntries(externalDB, query, caseSensitive, exactMatch)
			}

			// Add database context to results
			for _, entry := range results {
				allResults = append(allResults, SearchResult{
					Entry:        entry,
					DatabaseName: dbName,
					DatabasePath: dbPath,
				})
			}

			// Close the external database
			closeKeepassDB(externalDB)
		}

		if totalDBsSearched == 0 {
			fmt.Println("No accessible external databases found.")
			return
		}

		if len(allResults) == 0 {
			fmt.Printf("No entries found matching '%s' in %d database(s).\n", query, totalDBsSearched)
			return
		}

		fmt.Printf("Found %d entries matching '%s' across %d database(s):\n", len(allResults), query, totalDBsSearched)
		for _, result := range allResults {
			entry := result.Entry
			title := entry.GetTitle()
			username := getEntryValue(entry, "UserName")
			url := getEntryValue(entry, "URL")
			notes := getEntryValue(entry, "Notes")
			uuid := entry.UUID

			fmt.Printf("\n--- %s ---\n", title)
			fmt.Printf("Database: %s\n", result.DatabaseName)
			if username != "" {
				fmt.Printf("Username: %s\n", username)
			}
			if url != "" {
				fmt.Printf("URL: %s\n", url)
			}
			if notes != "" && len(notes) > 0 {
				// Truncate long notes for display
				if len(notes) > 100 {
					fmt.Printf("Notes: %s...\n", notes[:100])
				} else {
					fmt.Printf("Notes: %s\n", notes)
				}
			}
			fmt.Printf(("UUID: %x\n"), uuid)
		}
	},
}
