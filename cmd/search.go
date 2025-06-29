package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tobischo/gokeepasslib/v3"
)

func init() {
	rootCmd.AddCommand(searchCmd)

	// Add search flags
	searchCmd.Flags().BoolP("case-sensitive", "c", false, "Perform case-sensitive search")
	searchCmd.Flags().BoolP("exact", "e", false, "Exact match only (no fuzzy search)")
	searchCmd.Flags().StringP("group", "g", "", "Search only in specific group")
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

// fuzzySearchEntries performs fuzzy search across all groups and entries
func fuzzySearchEntries(db *gokeepasslib.Database, query string, caseSensitive bool, exactMatch bool) []gokeepasslib.Entry {
	var results []gokeepasslib.Entry
	if !caseSensitive {
		query = strings.ToLower(query)
	}

	// Search through all groups recursively
	for _, group := range db.Content.Root.Groups {
		results = append(results, searchEntriesInGroup(&group, query, caseSensitive, exactMatch)...)
	}

	return results
}

// searchEntriesInGroup searches entries within a specific group and its subgroups
func searchEntriesInGroup(group *gokeepasslib.Group, query string, caseSensitive bool, exactMatch bool) []gokeepasslib.Entry {
	var results []gokeepasslib.Entry

	// Search entries in current group
	for _, entry := range group.Entries {
		if fuzzyMatch(entry, query, caseSensitive, exactMatch) {
			results = append(results, entry)
		}
	}

	// Recursively search subgroups
	for _, subGroup := range group.Groups {
		results = append(results, searchEntriesInGroup(&subGroup, query, caseSensitive, exactMatch)...)
	}

	return results
}

// fuzzyMatch performs fuzzy matching on entry fields
func fuzzyMatch(entry gokeepasslib.Entry, query string, caseSensitive bool, exactMatch bool) bool {
	// Get searchable fields
	title := entry.GetTitle()
	username := getEntryValue(entry, "UserName")
	url := getEntryValue(entry, "URL")
	notes := getEntryValue(entry, "Notes")
	attributes := getAllEntryAttributes(entry)

	// Custom fields
	// dbPath := getEntryValue(entry, "Database Path")
	// dbType := getEntryValue(entry, "Database Type")

	// Convert to lowercase if not case sensitive
	if !caseSensitive {
		title = strings.ToLower(title)
		username = strings.ToLower(username)
		url = strings.ToLower(url)
		notes = strings.ToLower(notes)
		// dbPath = strings.ToLower(dbPath)
		// dbType = strings.ToLower(dbType)
	}

	if exactMatch {
		// Exact match only
		return title == query ||
			username == query ||
			url == query ||
			notes == query
		// dbPath == query ||
		// dbType == query
	}

	// Check for substring matches first
	if strings.Contains(title, query) ||
		strings.Contains(username, query) ||
		strings.Contains(url, query) ||
		strings.Contains(notes, query) ||
		strings.Contains(attributes, query) {
		// strings.Contains(dbPath, query) ||
		// strings.Contains(dbType, query) {
		return true
	}

	// Fuzzy matching: check if most characters from query appear in order
	return fuzzyStringMatch(title, query) ||
		fuzzyStringMatch(username, query) ||
		fuzzyStringMatch(url, query) ||
		fuzzyStringMatch(notes, query) ||
		fuzzyStringMatch(attributes, query)
	// fuzzyStringMatch(dbPath, query)
}

// getEntryValue safely gets a value from an entry
func getEntryValue(entry gokeepasslib.Entry, key string) string {
	for _, value := range entry.Values {
		if value.Key == key {
			return value.Value.Content
		}
	}
	return ""
}

func getAllEntryAttributes(entry gokeepasslib.Entry) string {
	var attributes []string
	for _, value := range entry.Values {
		if value.Key != "Title" && value.Key != "UserName" && value.Key != "URL" && value.Key != "Notes" && value.Key != "Password" {
			attributes = append(attributes, value.Value.Content)
		}
	}
	return strings.Join(attributes, " ")
}

// fuzzyStringMatch performs character-by-character fuzzy matching
func fuzzyStringMatch(text, pattern string) bool {
	if len(pattern) == 0 {
		return true
	}
	if len(text) == 0 {
		return false
	}

	// Simple fuzzy matching: check if pattern characters appear in order
	textIdx := 0
	patternIdx := 0

	for textIdx < len(text) && patternIdx < len(pattern) {
		if text[textIdx] == pattern[patternIdx] {
			patternIdx++
		}
		textIdx++
	}

	// If we matched all pattern characters, it's a fuzzy match
	return patternIdx == len(pattern)
}
