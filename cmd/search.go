package cmd

import (
	"fmt"
	"log"
	"os"
	"strconv"

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
	searchCmd.Flags().BoolP("favorites", "f", false, "Select entries for favorites")
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
		setFavorites, _ := cmd.Flags().GetBool("favorites")

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

		var allResults []SearchResult
		totalDBsSearched := 0

		for _, dbEntry := range databasesGroup.Entries {
			dbName := dbEntry.GetTitle()

			// If filter for specific database is set, skip all that don't match
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

			if _, err := os.Stat(dbPath); os.IsNotExist(err) {
				fmt.Printf("Warning: Database file '%s' not found for database '%s', skipping.\n", dbPath, dbName)
				continue
			}

			externalDB, err := openExternalKeepassDB(dbPath, dbPassword, keyFilePath)
			if err != nil {
				fmt.Printf("Warning: Failed to open database '%s': %v, skipping.\n", dbName, err)
				continue
			}

			totalDBsSearched++

			var results []gokeepasslib.Entry
			if targetGroup != "" {
				group := FindRootGroupByName(externalDB.Content.Root.Groups, targetGroup)
				if group != nil {
					results = searchEntriesInGroup(group, query, caseSensitive, exactMatch)
				}
			} else {
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

		selections := map[string]SearchResult{}
		fmt.Printf("Found %d entries matching '%s' across %d database(s):\n", len(allResults), query, totalDBsSearched)
		for count, result := range allResults {
			count++
			countStr := strconv.Itoa(count)
			if err != nil {
				fmt.Printf("Error converting string to int: %v\n", err)
				return
			}
			entry := result.Entry
			selections[countStr] = result
			printSearchResult(countStr, entry, result.DatabaseName)
		}
		if setFavorites {
			result, err := selectFavoriteEntry(selections)
			if err != nil {
				fmt.Printf("\nError: %v. Please try again.\n", err)
				return
			}
			fmt.Printf("\nSelected entry: %s (UUID: %x, DB: %s)\n", result.Entry.GetTitle(), result.Entry.UUID, result.DatabaseName)
			err = addFavoriteEntryToGoKP(gokpDB, result)
			if err != nil {
				fmt.Printf("\nError adding entry to favorites: %v\n", err)
				return
			}
			fmt.Println("Entry added to favorites successfully.")
			saveKeepassDB(gokpDB, gokpKDBX)
		}
	},
}
