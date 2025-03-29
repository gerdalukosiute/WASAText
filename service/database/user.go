package database

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

// GetOrCreateUser retrieves a user by name or creates a new one if it doesn't exist
func (db *appdbimpl) GetOrCreateUser(name string) (string, error) {
    // Validate username length and pattern before database operations
    if len(name) < 3 || len(name) > 16 {
        return "", ErrInvalidNameLength
    }

    namePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]{3,16}$`)
    if !namePattern.MatchString(name) {
        return "", ErrInvalidNameFormat
    }

    // First, try to get the user
    var userID string
    err := db.c.QueryRow("SELECT id FROM users WHERE name = ?", name).Scan(&userID)
    if err == nil {
        // User exists, return the ID
        return userID, nil
    }
    
    // If error is not "no rows", return the error
    if !errors.Is(err, sql.ErrNoRows) {
        return "", fmt.Errorf("error querying user: %w", err)
    }
    
    // User doesn't exist, create a new one with a 12-character identifier
    for attempts := 0; attempts < 5; attempts++ {
        userID = GenerateUserID()
        
        // Check if this ID is already used as a name 
        var count int
        err = db.c.QueryRow("SELECT COUNT(*) FROM users WHERE name = ?", userID).Scan(&count)
        if err != nil {
            return "", fmt.Errorf("error checking user ID: %w", err)
        }
        if count > 0 {
            // This ID is already used as a name, try another one
            continue
        }
        
        // Insert the new user
        _, err = db.c.Exec("INSERT INTO users (id, name) VALUES (?, ?)", userID, name)
        if err != nil {
            // Check for unique constraint violation
            var sqliteErr sqlite3.Error
            if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
                // Another concurrent request might have created the user, try to get it
                err = db.c.QueryRow("SELECT id FROM users WHERE name = ?", name).Scan(&userID)
                if err == nil {
                    return userID, nil
                }
                return "", ErrNameAlreadyTaken
            }
            return "", fmt.Errorf("error creating user: %w", err)
        }

        logrus.WithFields(logrus.Fields{
            "name": name,
            "id":   userID,
        }).Info("Created new user")

        return userID, nil
    }
    
    return "", fmt.Errorf("failed to generate a unique user ID after multiple attempts")
}

func (db *appdbimpl) UpdateUsername(userID string, newName string) error {
    // First check if the user exists
    var exists bool
    err := db.c.QueryRow("SELECT 1 FROM users WHERE id = ?", userID).Scan(&exists)
    if errors.Is(err, sql.ErrNoRows) {
        return ErrUserNotFound
    }
    if err != nil {
        return err
    }

    // Check if the new username is already taken by another user
    var existingUserID string
    err = db.c.QueryRow("SELECT id FROM users WHERE name = ?", newName).Scan(&existingUserID)
    if err != nil && !errors.Is(err, sql.ErrNoRows) {
        return err
    }
    
    // If a user with this name exists and is not the current user
    if err == nil && existingUserID != userID {
        return ErrDuplicateUsername
    }

    // Try to update the username
    _, err = db.c.Exec("UPDATE users SET name = ? WHERE id = ?", newName, userID)
    if err != nil {
        // Use errors.As instead of type assertion for checking
        var sqliteErr sqlite3.Error
        if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
            return ErrDuplicateUsername
        }
        return err
    }

    return nil
}

// UpdateUserPhoto updates the photo for a given user ID
func (db *appdbimpl) UpdateUserPhoto(userID string, fileData []byte, contentType string) (string, string, error) {
   logrus.WithFields(logrus.Fields{
       "userID": userID,
   }).Info("Updating user photo")

   // Start a transaction
   tx, err := db.c.Begin()
   if err != nil {
       return "", "", fmt.Errorf("error starting transaction: %w", err)
   }

   // Ensure transaction is rolled back if an error occurs
   defer func() {
       if tx != nil {
           if rollbackErr := tx.Rollback(); rollbackErr != nil {
               logrus.WithError(rollbackErr).Error("Error rolling back transaction")
           }
       }
   }()

   // Get the old photo ID
   var oldPhotoID sql.NullString
   err = tx.QueryRow("SELECT photo_id FROM users WHERE id = ?", userID).Scan(&oldPhotoID)
   if err != nil {
       if errors.Is(err, sql.ErrNoRows) {
           logrus.WithField("userID", userID).Error("User not found")
           return "", "", ErrUserNotFound
       }
       logrus.WithError(err).Error("Error querying user")
       return "", "", fmt.Errorf("error querying user: %w", err)
   }

   // Generate a unique photo ID
   photoID := db.GeneratePhotoID(userID)

   // Store the photo data directly in the media_files table
   _, err = tx.Exec(`
       INSERT INTO media_files (id, file_data, mime_type, created_at)
       VALUES (?, ?, ?, ?)
   `, photoID, fileData, contentType, time.Now())
   if err != nil {
       logrus.WithError(err).Error("Error storing photo data in database")
       return "", "", fmt.Errorf("error storing photo data: %w", err)
   }

   // Update the photo ID in the users table
   _, err = tx.Exec("UPDATE users SET photo_id = ? WHERE id = ?", photoID, userID)
   if err != nil {
       logrus.WithError(err).Error("Error updating user photo")
       return "", "", fmt.Errorf("error updating user photo: %w", err)
   }

   // Commit the transaction
   if err := tx.Commit(); err != nil {
       logrus.WithError(err).Error("Error committing transaction")
       return "", "", fmt.Errorf("error committing transaction: %w", err)
   }

   // Set tx to nil to prevent rollback in defer function
   tx = nil

   var oldPhotoIDString string
   if oldPhotoID.Valid {
       oldPhotoIDString = oldPhotoID.String
   }

   logrus.WithFields(logrus.Fields{
       "userID":     userID,
       "oldPhotoID": oldPhotoIDString,
       "newPhotoID": photoID,
   }).Info("User photo updated successfully")

   return oldPhotoIDString, photoID, nil
}

// Helper functions

// generateUserID creates a 12-character identifier following the pattern ^[a-zA-Z0-9_-]{12}$
func GenerateUserID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
	const idLength = 12
	
	// Initialize random source with current time
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	var sb strings.Builder
	sb.Grow(idLength)
	
	for i := 0; i < idLength; i++ {
		sb.WriteByte(charset[r.Intn(len(charset))])
	}
	
	return sb.String()
}

// isValidUserID checks if the user ID matches the required pattern
// Pattern: ^[a-zA-Z0-9_-]{12}$
func (db *appdbimpl) IsValidUserID(userID string) bool {
	if len(userID) != 12 {
		return false
	}

	for _, char := range userID {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-') {
			return false
		}
	}

	return true
}

// isValidImageType checks if the content type is a valid image type
func (db *appdbimpl) IsValidImageType(contentType string) bool {
	validTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
	}
	return validTypes[contentType]
}

// generatePhotoID generates a unique photo ID that matches the required pattern
// Pattern: ^[a-zA-Z0-9_-]{10,30}$
func (db *appdbimpl) GeneratePhotoID(userID string) string {
	// Create a timestamp-based ID with a random component
	timestamp := time.Now().UnixNano()
	randomPart := rand.Intn(1000000) // Add some randomness

	// Format: photo_[first 4 chars of userID]_[timestamp]_[random]
	// This ensures the ID is unique and matches the pattern
	photoID := fmt.Sprintf("photo_%s_%d_%d", userID[:4], timestamp, randomPart)

	// Ensure the ID is within the length limits (10-30 chars)
	if len(photoID) > 30 {
		photoID = photoID[:30]
	}

	return photoID
}