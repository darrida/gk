package cmd

import (
	"fmt"
	"os"

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
