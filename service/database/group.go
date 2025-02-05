package database

import (
	"database/sql"
	"fmt"
	"strings"
)

type AddUsersToGroupResult struct {
	AddedUsers  []string
	FailedUsers []string
}

// Update IsGroupMember to check both tables
func (db *appdbimpl) IsGroupMember(groupID, userID string) (bool, error) {
	var exists bool
	err := db.c.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM user_conversations uc
			JOIN conversations c ON uc.conversation_id = c.id
			WHERE c.id = ? AND uc.user_id = ? AND c.is_group = 1
		) AND EXISTS(
			SELECT 1 FROM group_members
			WHERE group_id = ? AND user_id = ?
		)
	`, groupID, userID, groupID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error checking group membership: %w", err)
	}
	return exists, nil
}

func (db *appdbimpl) AddUserToGroup(groupID, adderID, username string, title string) error {
	// Check if the group exists in conversations
	var groupExists bool
	err := db.c.QueryRow("SELECT EXISTS(SELECT 1 FROM conversations WHERE id = ? AND is_group = 1)", groupID).Scan(&groupExists)
	if err != nil {
		return fmt.Errorf("error checking group existence: %w", err)
	}

	// If the group doesn't exist, create it and add the adder as the first member
	if !groupExists {
		// Start a transaction
		tx, err := db.c.Begin()
		if err != nil {
			return fmt.Errorf("error starting transaction: %w", err)
		}
		defer tx.Rollback() // Rollback the transaction if it's not committed

		// Use the provided title or a default one
		if title == "" {
			title = fmt.Sprintf("Group %s", groupID[:8]) // Use first 8 characters of groupID as a default title
		}

		// Create the group in conversations table
		_, err = tx.Exec("INSERT INTO conversations (id, is_group, title) VALUES (?, 1, ?)", groupID, title)
		if err != nil {
			return fmt.Errorf("error creating group in conversations: %w", err)
		}

		// Create the group in groups table
		_, err = tx.Exec("INSERT INTO groups (id, name) VALUES (?, ?)", groupID, title)
		if err != nil {
			return fmt.Errorf("error creating group in groups table: %w", err)
		}

		// Add the adder as the first member of the group
		_, err = tx.Exec("INSERT INTO user_conversations (user_id, conversation_id) VALUES (?, ?)", adderID, groupID)
		if err != nil {
			return fmt.Errorf("error adding adder to new group: %w", err)
		}

		// Add the adder to group_members table
		_, err = tx.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?, ?)", groupID, adderID)
		if err != nil {
			return fmt.Errorf("error adding adder to group_members: %w", err)
		}

		// Commit the transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error committing transaction: %w", err)
		}
	} else {
		// If the group exists, check if the adder is a member
		isMember, err := db.IsGroupMember(groupID, adderID)
		if err != nil {
			return fmt.Errorf("error checking group membership: %w", err)
		}
		if !isMember {
			return ErrUnauthorized
		}
	}

	// Get the user ID for the given username
	var userID string
	err = db.c.QueryRow("SELECT id FROM users WHERE name = ?", username).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrUserNotFound
		}
		return fmt.Errorf("error getting user ID: %w", err)
	}

	// Check if the user is already a member of the group
	isMember, err := db.IsGroupMember(groupID, userID)
	if err != nil {
		return fmt.Errorf("error checking if user is already in group: %w", err)
	}
	if isMember {
		return ErrUserAlreadyInGroup
	}

	// Start a transaction for adding the new user
	tx, err := db.c.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction for adding user: %w", err)
	}
	defer tx.Rollback() // Rollback the transaction if it's not committed

	// Add the user to the group in user_conversations
	_, err = tx.Exec("INSERT INTO user_conversations (user_id, conversation_id) VALUES (?, ?)", userID, groupID)
	if err != nil {
		return fmt.Errorf("error adding user to user_conversations: %w", err)
	}

	// Add the user to the group in group_members
	_, err = tx.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?, ?)", groupID, userID)
	if err != nil {
		return fmt.Errorf("error adding user to group_members: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction for adding user: %w", err)
	}

	return nil
}

// not used yet
func (db *appdbimpl) GetUserByUsername(username string) (User, error) {
	var user User
	err := db.c.QueryRow("SELECT id, name FROM users WHERE name = ?", username).Scan(&user.ID, &user.Name)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

// Update LeaveGroup to remove from both tables
func (db *appdbimpl) LeaveGroup(groupID string, userID string) (username string, isGroupDeleted bool, err error) {
	// Check if the user is a member of the group
	isMember, err := db.IsGroupMember(groupID, userID)
	if err != nil {
		return "", false, fmt.Errorf("error checking group membership: %w", err)
	}
	if !isMember {
		return "", false, ErrUnauthorized
	}

	// Get the username
	err = db.c.QueryRow("SELECT name FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		return "", false, fmt.Errorf("error getting username: %w", err)
	}

	// Start a transaction
	tx, err := db.c.Begin()
	if err != nil {
		return "", false, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// Remove the user from the group in both tables
	_, err = tx.Exec("DELETE FROM user_conversations WHERE conversation_id = ? AND user_id = ?", groupID, userID)
	if err != nil {
		return "", false, fmt.Errorf("error removing user from user_conversations: %w", err)
	}
	_, err = tx.Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = ?", groupID, userID)
	if err != nil {
		return "", false, fmt.Errorf("error removing user from group_members: %w", err)
	}

	// Check if the group is empty
	var memberCount int
	err = tx.QueryRow("SELECT COUNT(*) FROM user_conversations WHERE conversation_id = ?", groupID).Scan(&memberCount)
	if err != nil {
		return "", false, fmt.Errorf("error checking group member count: %w", err)
	}

	if memberCount == 0 {
		// Delete the group from both tables
		_, err = tx.Exec("DELETE FROM conversations WHERE id = ?", groupID)
		if err != nil {
			return "", false, fmt.Errorf("error deleting empty group from conversations: %w", err)
		}
		_, err = tx.Exec("DELETE FROM groups WHERE id = ?", groupID)
		if err != nil {
			return "", false, fmt.Errorf("error deleting empty group from groups: %w", err)
		}
		isGroupDeleted = true
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return "", false, fmt.Errorf("error committing transaction: %w", err)
	}

	return username, isGroupDeleted, nil
}

// Update SetGroupName to update both tables
func (db *appdbimpl) SetGroupName(groupID string, userID string, newName string) (oldName string, updatedName string, err error) {
	// Check if the user is a member of the group
	isMember, err := db.IsGroupMember(groupID, userID)
	if err != nil {
		return "", "", fmt.Errorf("error checking group membership: %w", err)
	}
	if !isMember {
		return "", "", ErrUnauthorized
	}

	// Start a transaction
	tx, err := db.c.Begin()
	if err != nil {
		return "", "", fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the old group name
	err = tx.QueryRow("SELECT title FROM conversations WHERE id = ? AND is_group = 1", groupID).Scan(&oldName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", ErrGroupNotFound
		}
		return "", "", fmt.Errorf("error getting old group name: %w", err)
	}

	// Update the group name in both tables
	_, err = tx.Exec("UPDATE conversations SET title = ? WHERE id = ? AND is_group = 1", newName, groupID)
	if err != nil {
		return "", "", fmt.Errorf("error updating group name in conversations: %w", err)
	}
	_, err = tx.Exec("UPDATE groups SET name = ? WHERE id = ?", newName, groupID)
	if err != nil {
		return "", "", fmt.Errorf("error updating group name in groups: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return "", "", fmt.Errorf("error committing transaction: %w", err)
	}

	return oldName, newName, nil
}

// Update SetGroupPhoto to update both tables (if applicable)
func (db *appdbimpl) SetGroupPhoto(groupID string, userID string, newPhotoURL string) (oldPhotoURL string, updatedPhotoURL string, err error) {
	// Check if the user is a member of the group
	isMember, err := db.IsGroupMember(groupID, userID)
	if err != nil {
		return "", "", fmt.Errorf("error checking group membership: %w", err)
	}
	if !isMember {
		return "", "", ErrUnauthorized
	}

	// Start a transaction
	tx, err := db.c.Begin()
	if err != nil {
		return "", "", fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the old photo URL
	err = tx.QueryRow("SELECT COALESCE(profile_photo, '') FROM conversations WHERE id = ? AND is_group = 1", groupID).Scan(&oldPhotoURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", ErrGroupNotFound
		}
		return "", "", fmt.Errorf("error getting old group photo URL: %w", err)
	}

	// Update the group photo URL in conversations table
	_, err = tx.Exec("UPDATE conversations SET profile_photo = ? WHERE id = ? AND is_group = 1", newPhotoURL, groupID)
	if err != nil {
		return "", "", fmt.Errorf("error updating group photo URL in conversations: %w", err)
	}

	// Update the group photo URL in groups table if it has a profile_photo column
	// If the groups table doesn't have a profile_photo column, you can skip this part
	_, err = tx.Exec("UPDATE groups SET profile_photo = ? WHERE id = ?", newPhotoURL, groupID)
	if err != nil {
		// If the error is due to the column not existing, we can ignore it
		// Otherwise, return the error
		if !strings.Contains(err.Error(), "no such column: profile_photo") {
			return "", "", fmt.Errorf("error updating group photo URL in groups: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return "", "", fmt.Errorf("error committing transaction: %w", err)
	}

	return oldPhotoURL, newPhotoURL, nil
}

func (db *appdbimpl) UserExists(userID string) (bool, error) {
	var exists bool
	err := db.c.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error checking user existence: %w", err)
	}
	return exists, nil
}
