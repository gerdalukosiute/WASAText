package database

import (
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

// GetOrCreateUser retrieves a user by name or creates a new one if it doesn't exist
func (db *appdbimpl) GetOrCreateUser(name string) (string, error) {
	// Validate username length before database operations
	if len(name) < 3 || len(name) > 16 {
		return "", fmt.Errorf("invalid username length")
	}

	var userID string
	err := db.c.QueryRow("SELECT id FROM users WHERE name = ?", name).Scan(&userID)
	if err == sql.ErrNoRows {
		// User doesn't exist, create a new one
		userID = uuid.Must(uuid.NewV4()).String()
		_, err = db.c.Exec("INSERT INTO users (id, name) VALUES (?, ?)", userID, name)
		if err != nil {
			return "", fmt.Errorf("error creating user: %w", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("error querying user: %w", err)
	}
	return userID, nil
}

func (db *appdbimpl) UpdateUsername(userID string, newName string) error {
	// First check if the user exists
	var exists bool
	err := db.c.QueryRow("SELECT 1 FROM users WHERE id = ?", userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}

	// Try to update the username
	_, err = db.c.Exec("UPDATE users SET name = ? WHERE id = ?", newName, userID)
	if err != nil {
		// Check for unique constraint violation
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.Code == sqlite3.ErrConstraint {
			return ErrDuplicateUsername
		}
		return err
	}

	return nil
}

// UpdateUserPhoto updates the photo URL for a given user ID
func (db *appdbimpl) UpdateUserPhoto(userID string, photoURL string) (string, error) {
	logrus.WithFields(logrus.Fields{
		"userID":   userID,
		"photoURL": photoURL,
	}).Info("Updating user photo")

	var oldPhotoURL sql.NullString
	err := db.c.QueryRow("SELECT photo_url FROM users WHERE id = ?", userID).Scan(&oldPhotoURL)
	if err == sql.ErrNoRows {
		logrus.WithField("userID", userID).Error("User not found")
		return "", ErrUserNotFound
	}
	if err != nil {
		logrus.WithError(err).Error("Error querying user")
		return "", err
	}

	_, err = db.c.Exec("UPDATE users SET photo_url = ? WHERE id = ?", photoURL, userID)
	if err != nil {
		logrus.WithError(err).Error("Error updating user photo")
		return "", err
	}

	var oldPhotoURLString string
	if oldPhotoURL.Valid {
		oldPhotoURLString = oldPhotoURL.String
	}

	logrus.WithFields(logrus.Fields{
		"userID":      userID,
		"oldPhotoURL": oldPhotoURLString,
		"newPhotoURL": photoURL,
	}).Info("User photo updated successfully")

	return oldPhotoURLString, nil
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
