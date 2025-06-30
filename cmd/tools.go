package cmd

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/tobischo/gokeepasslib/v3"
	"golang.org/x/term"
)

func addGoKPEntryToGroup(db *gokeepasslib.Database, groupName string, title string, password string, path string, key string) {
	entry := gokeepasslib.NewEntry()

	// Standard fields
	entry.Values = append(entry.Values, mkValue("Title", title))
	entry.Values = append(entry.Values, mkValue("UserName", ""))
	entry.Values = append(entry.Values, mkProtectedValue("Password", password))
	// entry.Values = append(entry.Values, mkValue("URL", key))

	// Additional database-specific attributes
	entry.Values = append(entry.Values, mkValue("Database Path", path))
	entry.Values = append(entry.Values, mkValue("Key File Path", key))
	entry.Values = append(entry.Values, mkValue("Database Type", "KeePass"))
	entry.Values = append(entry.Values, mkValue("Format", "KDBX"))
	entry.Values = append(entry.Values, mkValue("Created Date", getCurrentTimestamp("datetime")))
	entry.Values = append(entry.Values, mkValue("Last Modified", getCurrentTimestamp("iso")))
	entry.Values = append(entry.Values, mkValue("Notes", "External KeePass database managed by gokp"))

	for i := range db.Content.Root.Groups {
		if db.Content.Root.Groups[i].Name == groupName {
			db.Content.Root.Groups[i].Entries = append(db.Content.Root.Groups[i].Entries, entry)
			break
		}
	}
}

func addFavoriteEntryToGoKP(db *gokeepasslib.Database, result SearchResult) error {
	// Find favorites entry index value in root group
	var favGroupIndex int
	for i := range db.Content.Root.Groups {
		if db.Content.Root.Groups[i].Name == "favorites" {
			favGroupIndex = i
			break
		}
	}
	// Loop over entries in favorites group to find existing indexes
	maxIndex := 1
	for _, entry := range db.Content.Root.Groups[favGroupIndex].Entries {
		for _, value := range entry.Values {
			if entry.UUID == result.Entry.UUID {
				return fmt.Errorf("entry '%s' already exists in favorites", entry.GetTitle())
			}
			index := value.Key == "Favorite Index"
			if index {
				indexValue, err := strconv.Atoi(value.Value.Content)
				if err != nil {
					return fmt.Errorf("error converting index value to int: %v", err)
				}
				if indexValue > maxIndex {
					maxIndex = indexValue
				}
			}
		}
	}

	newEntry := gokeepasslib.NewEntry()

	entry := result.Entry
	title := entry.GetTitle()
	username := getEntryValue(entry, "UserName")
	password := entry.GetPassword()
	if password == "" {
		return fmt.Errorf("entry '%s' has no password set, cannot add to favorites", title)
	}
	url := getEntryValue(entry, "URL")
	uuid := entry.UUID
	favIndex := maxIndex + 1

	// Create new favoties entry using the next favorites index value
	newEntry.Values = append(newEntry.Values, mkValue("Title", title))
	newEntry.Values = append(newEntry.Values, mkValue("UserName", username))
	newEntry.Values = append(newEntry.Values, mkProtectedValue("Password", password))
	newEntry.Values = append(newEntry.Values, mkValue("URL", url))

	// Additional database-specific attributes
	newEntry.Values = append(newEntry.Values, mkValue("Database Source", result.DatabaseName))
	newEntry.Values = append(newEntry.Values, mkValue("Database path", result.DatabasePath))
	newEntry.Values = append(newEntry.Values, mkValue("Database UUID", fmt.Sprintf("%x", uuid)))
	newEntry.Values = append(newEntry.Values, mkValue("Favorite Index", strconv.Itoa(favIndex)))
	newEntry.Values = append(newEntry.Values, mkValue("Created Date", getCurrentTimestamp("datetime")))
	newEntry.Values = append(newEntry.Values, mkValue("Last Modified", getCurrentTimestamp("iso")))
	newEntry.Values = append(newEntry.Values, mkValue("Notes", "Entry from external KeePass database managed by gokp"))

	db.Content.Root.Groups[favGroupIndex].Entries = append(db.Content.Root.Groups[favGroupIndex].Entries, newEntry)
	return nil
}

func readFavoritesEntry(db *gokeepasslib.Database, favIndex int) *gokeepasslib.Entry {
	var favGroupIndex int
	for i := range db.Content.Root.Groups {
		if db.Content.Root.Groups[i].Name == "favorites" {
			favGroupIndex = i
			break
		}
	}

	// Loop over entries in favorites group to find existing indexes
	for _, entry := range db.Content.Root.Groups[favGroupIndex].Entries {
		fmt.Println(entry.GetTitle(), entry.GetContent("Favorite Index"))
		if entry.GetContent("Favorite Index") == strconv.Itoa(favIndex) {
			return &entry
		}
	}
	return nil
}

func addEntryToGroup(db *gokeepasslib.Database, groupName string, title string, username string, password string, url string) {
	entry := gokeepasslib.NewEntry()
	entry.Values = append(entry.Values, mkValue("Title", title))
	entry.Values = append(entry.Values, mkValue("UserName", username))
	entry.Values = append(entry.Values, mkProtectedValue("Password", password))
	entry.Values = append(entry.Values, mkValue("URL", url))

	for i := range db.Content.Root.Groups {
		if db.Content.Root.Groups[i].Name == groupName {
			db.Content.Root.Groups[i].Entries = append(db.Content.Root.Groups[i].Entries, entry)
			// db.Content.Root.Groups[i] =
			break
		}
	}
}

func readEntryFromGroup(db *gokeepasslib.Database, groupName string, title string) *gokeepasslib.Entry {
	for i := range db.Content.Root.Groups {
		if db.Content.Root.Groups[i].Name == groupName {
			for j := range db.Content.Root.Groups[i].Entries {
				if db.Content.Root.Groups[i].Entries[j].GetTitle() == title {
					return &db.Content.Root.Groups[i].Entries[j]
				}
			}
			break
		}
	}
	return nil
}

func openKeepassDB(dbPath string, password string) (*gokeepasslib.Database, error) {
	file, err := os.Open(dbPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	db := gokeepasslib.NewDatabase()
	db.Credentials = gokeepasslib.NewPasswordCredentials(password)

	err = gokeepasslib.NewDecoder(file).Decode(db)
	if err != nil {
		return nil, err
	}

	if err := db.UnlockProtectedEntries(); err != nil {
		return nil, err
	}

	return db, nil
}

func closeKeepassDB(db *gokeepasslib.Database) {
	if db != nil {
		db.LockProtectedEntries()
	}
}

func saveKeepassDB(db *gokeepasslib.Database, dbPath string) error {
	writeFile, err := os.Create(dbPath)
	if err != nil {
		return err
	}
	defer writeFile.Close()

	db.LockProtectedEntries()
	keepassEncoder := gokeepasslib.NewEncoder(writeFile)
	if err := keepassEncoder.Encode(db); err != nil {
		return err
	}

	return nil
}

func getGoKPPassword() (string, error) {
	secret, err := get_password("gokp", "local")
	if err != nil {
		fmt.Print("Enter admin password: ")
		password, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", fmt.Errorf("failed to read password from terminal: %w", err)
		}
		if string(password) == "" {
			fmt.Println("\nPassword is required")
			os.Exit(1)
		}
		fmt.Println()
		secret = string(password)
	}
	return secret, nil
}

// Helper function to get any attribute value from an entry
func getEntryAttribute(entry *gokeepasslib.Entry, attributeName string) string {
	if entry == nil {
		return ""
	}

	for _, value := range entry.Values {
		if value.Key == attributeName {
			return value.Value.Content
		}
	}
	return ""
}

// Helper function to get formatted timestamps for different use cases
func getCurrentTimestamp(format string) string {
	now := time.Now()
	switch format {
	case "date":
		return now.Format("2006-01-02")
	case "datetime":
		return now.Format("2006-01-02 15:04:05")
	case "iso":
		return now.Format(time.RFC3339)
	case "readable":
		return now.Format("January 2, 2006 at 3:04 PM")
	default:
		return now.Format("2006-01-02 15:04:05")
	}
}
