package database

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/sirupsen/logrus"
)

// StoreMediaFile stores a media file in the database and returns its ID
func (db *appdbimpl) StoreMediaFile(fileData []byte, mimeType string) (string, error) {
	// Try up to 10 times to generate a unique ID
	for i := 0; i < 10; i++ {
		// Generate a timestamp-based ID with a prefix
		// Format: media + timestamp (nanoseconds)
		// This ensures IDs are between 10-30 characters
		timestamp := time.Now().UnixNano()
		mediaID := fmt.Sprintf("media%d", timestamp)

		// Ensure the ID length is between 10 and 30 characters
		if len(mediaID) < 10 {
			// This is unlikely to happen, but just in case
			mediaID = fmt.Sprintf("media%010d", timestamp)
		} else if len(mediaID) > 30 {
			// If too long, truncate but keep uniqueness
			mediaID = fmt.Sprintf("media%s", fmt.Sprint(timestamp)[0:20])
		}

		// Verify the ID matches the required pattern (media + numbers)
		if !regexp.MustCompile(`^media[0-9]{5,25}$`).MatchString(mediaID) {
			continue // Try again if pattern doesn't match
		}

		// Check if this ID already exists
		var exists bool
		err := db.c.QueryRow("SELECT EXISTS(SELECT 1 FROM media_files WHERE id = ?)", mediaID).Scan(&exists)
		if err != nil {
			return "", fmt.Errorf("error checking media ID existence: %w", err)
		}

		// If the ID already exists, try again
		if exists {
			time.Sleep(1 * time.Millisecond) // Small delay to ensure different timestamp
			continue
		}

		// Start a transaction
		tx, err := db.c.Begin()
		if err != nil {
			return "", fmt.Errorf("error starting transaction: %w", err)
		}

		// Ensure transaction is rolled back if an error occurs
		defer func() {
			if tx != nil {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					logrus.WithError(rollbackErr).Error("Error rolling back transaction")
				}
			}
		}()

		// Insert the media file
		_, err = tx.Exec(`
			INSERT INTO media_files (id, file_data, mime_type, created_at)
			VALUES (?, ?, ?, ?)
		`, mediaID, fileData, mimeType, time.Now())

		if err != nil {
			return "", fmt.Errorf("error storing media file: %w", err)
		}

		// Commit the transaction
		if err = tx.Commit(); err != nil {
			return "", fmt.Errorf("error committing transaction: %w", err)
		}

		// Set tx to nil to prevent rollback in defer function
		tx = nil

		return mediaID, nil
	}

	// If impossible to generate
	return "", fmt.Errorf("failed to generate a unique media ID after multiple attempts")
}

// GetMediaFile retrieves a media file from the database by its ID
func (db *appdbimpl) GetMediaFile(mediaID string) ([]byte, string, error) {
	var fileData []byte
	var mimeType string

	err := db.c.QueryRow(`
		SELECT file_data, mime_type FROM media_files WHERE id = ?
	`, mediaID).Scan(&fileData, &mimeType)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", fmt.Errorf("media file not found: %w", err)
		}
		return nil, "", fmt.Errorf("error retrieving media file: %w", err)
	}

	return fileData, mimeType, nil
}
