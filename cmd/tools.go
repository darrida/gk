package cmd

import (
	"fmt"
	"os"
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

func addEntryToGroupWithExtras(db *gokeepasslib.Database, groupName string, title string, username string, password string, url string, notes string) {
	entry := gokeepasslib.NewEntry()

	// Standard KeePass fields
	entry.Values = append(entry.Values, mkValue("Title", title))
	entry.Values = append(entry.Values, mkValue("UserName", username))
	entry.Values = append(entry.Values, mkProtectedValue("Password", password))
	entry.Values = append(entry.Values, mkValue("URL", url))
	entry.Values = append(entry.Values, mkValue("Notes", notes))

	// Additional custom attributes
	// if email != "" {
	// 	entry.Values = append(entry.Values, mkValue("Email", email))
	// }
	// if tags != "" {
	// 	entry.Values = append(entry.Values, mkValue("Tags", tags))
	// }

	// // Protected custom fields (encrypted)
	// entry.Values = append(entry.Values, mkProtectedValue("API Key", "secret-api-key"))
	// entry.Values = append(entry.Values, mkProtectedValue("SSH Private Key", "-----BEGIN RSA PRIVATE KEY-----"))
	// entry.Values = append(entry.Values, mkProtectedValue("Secret Token", "bearer-token-here"))

	for i := range db.Content.Root.Groups {
		if db.Content.Root.Groups[i].Name == groupName {
			db.Content.Root.Groups[i].Entries = append(db.Content.Root.Groups[i].Entries, entry)
			break
		}
	}
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

// Helper function to set/update any attribute in an entry
func setEntryAttribute(entry *gokeepasslib.Entry, attributeName string, attributeValue string, protected bool) {
	if entry == nil {
		return
	}

	// Check if attribute already exists and update it
	for i := range entry.Values {
		if entry.Values[i].Key == attributeName {
			if protected {
				entry.Values[i] = mkProtectedValue(attributeName, attributeValue)
			} else {
				entry.Values[i] = mkValue(attributeName, attributeValue)
			}
			return
		}
	}

	// If attribute doesn't exist, add it
	if protected {
		entry.Values = append(entry.Values, mkProtectedValue(attributeName, attributeValue))
	} else {
		entry.Values = append(entry.Values, mkValue(attributeName, attributeValue))
	}
}

// Function to list all attributes of an entry
func listEntryAttributes(entry *gokeepasslib.Entry) map[string]string {
	attributes := make(map[string]string)
	if entry == nil {
		return attributes
	}

	for _, value := range entry.Values {
		attributes[value.Key] = value.Value.Content
	}
	return attributes
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

func searchEntries(db *gokeepasslib.Database, query string) []*gokeepasslib.Entry {
	var results []*gokeepasslib.Entry

	for _, group := range db.Content.Root.Groups {
		for _, entry := range group.Entries {
			if entry.GetTitle() == query || entry.GetContent("UserName") == query || entry.GetContent("URL") == query {
				results = append(results, &entry)
			}
		}
	}

	return results
}
