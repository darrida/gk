package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/tobischo/gokeepasslib/v3"
)

// SearchResult represents a search result with database context
type SearchResult struct {
	Entry        gokeepasslib.Entry
	DatabaseName string
	DatabasePath string
}

// openExternalKeepassDB opens an external KeePass database with credentials and optional key file
func openExternalKeepassDB(dbPath, password, keyFilePath string) (*gokeepasslib.Database, error) {
	file, err := os.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database file '%s': %w", dbPath, err)
	}
	defer file.Close()

	db := gokeepasslib.NewDatabase()

	// Set up credentials
	if keyFilePath != "" {
		// Check if key file exists
		if _, err := os.Stat(keyFilePath); err != nil {
			return nil, fmt.Errorf("key file '%s' not found: %w", keyFilePath, err)
		}
		// Use both password and key file
		credentials, err := gokeepasslib.NewPasswordAndKeyCredentials(password, keyFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create credentials with key file: %w", err)
		}
		db.Credentials = credentials
	} else {
		// Use only password
		db.Credentials = gokeepasslib.NewPasswordCredentials(password)
	}

	err = gokeepasslib.NewDecoder(file).Decode(db)
	if err != nil {
		return nil, fmt.Errorf("failed to decode database '%s': %w", dbPath, err)
	}

	if err := db.UnlockProtectedEntries(); err != nil {
		return nil, fmt.Errorf("failed to unlock protected entries in '%s': %w", dbPath, err)
	}

	return db, nil
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

func printSearchResult(count string, entry gokeepasslib.Entry, databaseName string) {
	title := entry.GetTitle()
	username := getEntryValue(entry, "UserName")
	url := getEntryValue(entry, "URL")
	notes := getEntryValue(entry, "Notes")
	uuid := entry.UUID

	fmt.Printf("\n--------------------\n")
	if count != "" {
		fmt.Printf("Selection: %s\n", count)
		fmt.Printf("--------------------\n")
	}
	fmt.Printf("Title:     %s\n", title)
	fmt.Printf("Database:  %s\n", databaseName)
	if username != "" {
		fmt.Printf("Username:  %s\n", username)
	}
	if url != "" {
		fmt.Printf("URL:       %s\n", url)
	}
	if notes != "" && len(notes) > 0 {
		if len(notes) > 100 {
			fmt.Printf("Notes:     %s...\n", notes[:100])
		} else {
			fmt.Printf("Notes:     %s\n", notes)
		}
	}
	fmt.Printf(("UUID:      %x\n"), uuid)

	var firstPass bool = true
	for _, value := range entry.Values {
		if value.Key != "Password" && value.Key != "UserName" && value.Key != "URL" && value.Key != "Notes" && value.Key != "Title" {
			if firstPass {
				fmt.Println("Custom Attributes:")
				firstPass = false
			}
			fmt.Printf("- %s: %s\n", value.Key, value.Value.Content)
		}
	}
}

func selectFavoriteEntry(selections map[string]SearchResult) (SearchResult, error) {
	fmt.Printf("\n--------------------\n")
	fmt.Printf("SUMMARY SELECTION LIST:")
	for selector, result := range selections {
		fmt.Printf("\n%s: %s (UUID: %x, DB: %s)", selector, result.Entry.GetTitle(), result.Entry.UUID, result.DatabaseName)
	}

	fmt.Print("\n\nEntry number for entry to save:\n> ")
	var selected string
	fmt.Scanln(&selected)

	if selected == "" {
		// fmt.Println("\nNo input provided, exiting.")
		return SearchResult{}, fmt.Errorf("no input provided")
	}

	result := selections[selected]
	if result.Entry.GetTitle() == "" {
		// fmt.Printf("\nNo entry found for selection '%s'. Please try again.\n", selected)
		return SearchResult{}, fmt.Errorf("no entry found for selection '%s'", selected)
	}

	return result, nil
}
