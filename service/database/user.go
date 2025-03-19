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
	userID = generateUserID()
	
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

// generateUserID creates a 12-character identifier following the pattern ^[a-zA-Z0-9_-]{12}$
func generateUserID() string {
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

// UpdateUserPhoto updates the photo ID for a given user ID
func (db *appdbimpl) UpdateUserPhoto(userID string, photoID string) (string, error) {
	logrus.WithFields(logrus.Fields{
		"userID":  userID,
		"photoID": photoID,
	}).Info("Updating user photo")

	// Start a transaction
	tx, err := db.c.Begin()
	if err != nil {
		return "", fmt.Errorf("error starting transaction: %w", err)
	}

	// Get the old photo ID
	var oldPhotoID sql.NullString
	err = tx.QueryRow("SELECT photo_id FROM users WHERE id = ?", userID).Scan(&oldPhotoID)
	if err != nil {
		// Rollback the transaction and check for errors
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			logrus.WithError(rollbackErr).Error("Error rolling back transaction")
			return "", fmt.Errorf("error rolling back transaction: %w (original error: %w)", rollbackErr, err)
		}

		if errors.Is(err, sql.ErrNoRows) {
			logrus.WithField("userID", userID).Error("User not found")
			return "", ErrUserNotFound
		}
		logrus.WithError(err).Error("Error querying user")
		return "", fmt.Errorf("error querying user: %w", err)
	}

	// Update the photo ID
	_, err = tx.Exec("UPDATE users SET photo_id = ? WHERE id = ?", photoID, userID)
	if err != nil {
		// Rollback the transaction and check for errors
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			logrus.WithError(rollbackErr).Error("Error rolling back transaction")
			return "", fmt.Errorf("error rolling back transaction: %w (original error: %w)", rollbackErr, err)
		}

		logrus.WithError(err).Error("Error updating user photo")
		return "", fmt.Errorf("error updating user photo: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		logrus.WithError(err).Error("Error committing transaction")
		return "", fmt.Errorf("error committing transaction: %w", err)
	}

	var oldPhotoIDString string
	if oldPhotoID.Valid {
		oldPhotoIDString = oldPhotoID.String
	}

	logrus.WithFields(logrus.Fields{
		"userID":     userID,
		"oldPhotoID": oldPhotoIDString,
		"newPhotoID": photoID,
	}).Info("User photo updated successfully")

	return oldPhotoIDString, nil
}

// GetUserNameByID retrieves a user's name by their ID
func (db *appdbimpl) GetUserNameByID(userID string) (string, error) {
	var name string
	err := db.c.QueryRow("SELECT name FROM users WHERE id = ?", userID).Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrUserNotFound
		}
		return "", fmt.Errorf("error getting user name: %w", err)
	}
	return name, nil
}
