package database

import (
	"database/sql"
	"fmt"
	"strings"
	"errors"
	"time"
	"regexp"

	"github.com/sirupsen/logrus"
)

// Updated
func (db *appdbimpl) AddUsersToGroup(groupID, adderID string, usernames []string) (*GroupAddResult, error) {
	// First check if the conversation exists at all
	var conversationExists bool
	err := db.c.QueryRow("SELECT EXISTS(SELECT 1 FROM conversations WHERE id = ?)", groupID).Scan(&conversationExists)
	if err != nil {
		return nil, fmt.Errorf("error checking conversation existence: %w", err)
	}
	if !conversationExists {
		return nil, ErrGroupNotFound
	}


	// Now check if it's a group conversation
	var isGroup bool
	var currentGroupName string
	err = db.c.QueryRow("SELECT is_group, COALESCE(title, '') FROM conversations WHERE id = ?", groupID).Scan(&isGroup, &currentGroupName)
	if err != nil {
		return nil, fmt.Errorf("error getting conversation details: %w", err)
	}
	if !isGroup {
		return nil, ErrGroupNotFound
	}


	// Check if the adder is a member of the group
	var isMember bool
	err = db.c.QueryRow("SELECT EXISTS(SELECT 1 FROM user_conversations WHERE conversation_id = ? AND user_id = ?)", groupID, adderID).Scan(&isMember)
	if err != nil {
		return nil, fmt.Errorf("error checking adder membership: %w", err)
	}
	if !isMember {
		return nil, ErrUnauthorized
	}


	// Get the adder's name
	adderName, err := db.GetUserNameByID(adderID)
	if err != nil {
		return nil, fmt.Errorf("error getting adder name: %w", err)
	}


	// Start a transaction
	tx, err := db.c.Begin()
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	
	// Ensure transaction is rolled back if an error occurs
	defer func() {
		if tx != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				logrus.WithError(rollbackErr).Error("Error rolling back transaction")
			}
		}
	}()


	// Prepare result
	result := &GroupAddResult{
		GroupID:           groupID,
		GroupName:         currentGroupName,
		AddedUsers:        []struct {
			Username string
			UserID   string
		}{},
		FailedUsers:       []string{},
		AddedBy: User{
			ID:   adderID,
			Name: adderName,
		},
		Timestamp:         time.Now(),
	}


	// Process each username
	for _, username := range usernames {
		// Get the user ID for the given username
		var userID string
		err = tx.QueryRow("SELECT id FROM users WHERE name = ?", username).Scan(&userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// User not found, add to failed users
				result.FailedUsers = append(result.FailedUsers, username)
				continue
			}
			return nil, fmt.Errorf("error getting user ID: %w", err)
		}


		// Check if the user is already a member of the group
		var userExists bool
		err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM user_conversations WHERE conversation_id = ? AND user_id = ?)", groupID, userID).Scan(&userExists)
		if err != nil {
			return nil, fmt.Errorf("error checking user membership: %w", err)
		}
		if userExists {
			// User already in group, add to failed users
			result.FailedUsers = append(result.FailedUsers, username)
			continue
		}


		// Add the user to the conversation
		_, err = tx.Exec("INSERT INTO user_conversations (user_id, conversation_id) VALUES (?, ?)", userID, groupID)
		if err != nil {
			return nil, fmt.Errorf("error adding user to conversation: %w", err)
		}


		// Add the user to the group_members table if it exists
		_, err = tx.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?, ?)", groupID, userID)
		if err != nil {
			// If this fails, it might be because the group_members table is not used or the group_id doesn't exist there
			// We'll log the error but continue since the user was added to user_conversations
			logrus.WithError(err).Warnf("Failed to add user %s to group_members table", username)
		}


		// Add to successful users
		result.AddedUsers = append(result.AddedUsers, struct {
			Username string
			UserID   string
		}{
			Username: username,
			UserID:   userID,
		})
	}


	// Get updated member count
	var memberCount int
	err = tx.QueryRow("SELECT COUNT(*) FROM user_conversations WHERE conversation_id = ?", groupID).Scan(&memberCount)
	if err != nil {
		return nil, fmt.Errorf("error getting member count: %w", err)
	}
	result.UpdatedMemberCount = memberCount


	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}
	
	// Set tx to nil to prevent rollback in defer function
	tx = nil


	return result, nil
}

// Updated
func (db *appdbimpl) LeaveGroup(groupID string, userID string) (username string, isGroupDeleted bool, remainingMemberCount int, err error) {
  // Check if the user is a member of the group
 	isMember, err := db.IsGroupMember(groupID, userID)
 	if err != nil {
     	// If the error is that the group doesn't exist, return that specific error
     	if errors.Is(err, ErrGroupNotFound) {
        	return "", false, 0, ErrGroupNotFound
     	}
     	return "", false, 0, fmt.Errorf("error checking group membership: %w", err)
 	}
 	if !isMember {
     	return "", false, 0, ErrUnauthorized
 	}


  // Get the username
  var name string
  err = db.c.QueryRow("SELECT name FROM users WHERE id = ?", userID).Scan(&name)
  if err != nil {
      return "", false, 0, fmt.Errorf("error getting username: %w", err)
  }


  // Start a transaction
  tx, err := db.c.Begin()
  if err != nil {
      return "", false, 0, fmt.Errorf("error starting transaction: %w", err)
  }
 
  // Ensure transaction is rolled back if an error occurs
  defer func() {
      if tx != nil {
          if rollbackErr := tx.Rollback(); rollbackErr != nil {
              logrus.WithError(rollbackErr).Error("Error rolling back transaction")
          }
      }
  }()


  // Remove the user from the group in both tables
  _, err = tx.Exec("DELETE FROM user_conversations WHERE conversation_id = ? AND user_id = ?", groupID, userID)
  if err != nil {
      return "", false, 0, fmt.Errorf("error removing user from user_conversations: %w", err)
  }
 
  _, err = tx.Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = ?", groupID, userID)
  if err != nil {
      return "", false, 0, fmt.Errorf("error removing user from group_members: %w", err)
  }


  // Check if the group is empty
  var memberCount int
  err = tx.QueryRow("SELECT COUNT(*) FROM user_conversations WHERE conversation_id = ?", groupID).Scan(&memberCount)
  if err != nil {
      return "", false, 0, fmt.Errorf("error checking group member count: %w", err)
  }


  if memberCount == 0 {
      // Delete the group from both tables
      _, err = tx.Exec("DELETE FROM conversations WHERE id = ?", groupID)
      if err != nil {
          return "", false, 0, fmt.Errorf("error deleting empty group from conversations: %w", err)
      }
     
      _, err = tx.Exec("DELETE FROM groups WHERE id = ?", groupID)
      if err != nil {
          return "", false, 0, fmt.Errorf("error deleting empty group from groups: %w", err)
      }
     
      isGroupDeleted = true
  }


  // Commit the transaction
  if err := tx.Commit(); err != nil {
      return "", false, 0, fmt.Errorf("error committing transaction: %w", err)
  }


  // Set tx to nil to prevent rollback in defer function
  tx = nil


  return name, isGroupDeleted, memberCount, nil
}

// Updated
func (db *appdbimpl) IsGroupMember(groupID string, userID string) (bool, error) {
  // First check if the group exists
  var groupExists int
  err := db.c.QueryRow(`
      SELECT COUNT(*)
      FROM conversations
      WHERE id = ? AND is_group = 1
  `, groupID).Scan(&groupExists)
 
  if err != nil {
      return false, fmt.Errorf("error checking group existence: %w", err)
  }
 
  if groupExists == 0 {
      return false, ErrGroupNotFound
  }
 
  // Now check if the user is a member in the user_conversations table
  var isInUserConversations int
  err = db.c.QueryRow(`
      SELECT COUNT(*)
      FROM user_conversations
      WHERE conversation_id = ? AND user_id = ?
  `, groupID, userID).Scan(&isInUserConversations)
 
  if err != nil {
      return false, fmt.Errorf("error checking user_conversations membership: %w", err)
  }
 
  // Also check the group_members table for consistency
  var isInGroupMembers int
  err = db.c.QueryRow(`
      SELECT COUNT(*)
      FROM group_members
      WHERE group_id = ? AND user_id = ?
  `, groupID, userID).Scan(&isInGroupMembers)
 
  if err != nil {
      // If there's an error with group_members, log it but rely on user_conversations
      logrus.WithError(err).Warn("Error checking group_members table, using user_conversations result")
      return isInUserConversations > 0, nil
  }
 
  // If the user is in both tables, they're definitely a member
  if isInUserConversations > 0 && isInGroupMembers > 0 {
      return true, nil
  }
 
  // If there's a discrepancy between the tables, log it
  if isInUserConversations != isInGroupMembers {
      logrus.WithFields(logrus.Fields{
          "groupID": groupID,
          "userID": userID,
          "inUserConversations": isInUserConversations > 0,
          "inGroupMembers": isInGroupMembers > 0,
      }).Warn("Inconsistency between user_conversations and group_members tables")
  }
 
  // Use user_conversations as the source of truth
  return isInUserConversations > 0, nil
}

// Updated 
func (db *appdbimpl) SetGroupName(groupID string, userID string, newName string) (oldName string, updatedName string, memberCount int, err error) {
   // Validate the new group name format
   if len(newName) < 3 || len(newName) > 30 {
       return "", "", 0, ErrInvalidGroupName
   }
  
   // Check if the group name matches the required pattern 
   validNamePattern := "^[a-zA-Z0-9_\\s-]{3,30}$"
   match, err := regexp.MatchString(validNamePattern, newName)
   if err != nil {
       return "", "", 0, fmt.Errorf("error validating group name: %w", err)
   }
   if !match {
       return "", "", 0, ErrInvalidGroupName
   }
  
   // Check if the user is a member of the group
   isMember, err := db.IsGroupMember(groupID, userID)
   if err != nil {
       if errors.Is(err, ErrGroupNotFound) {
           return "", "", 0, ErrGroupNotFound
       }
       return "", "", 0, fmt.Errorf("error checking group membership: %w", err)
   }
   if !isMember {
       return "", "", 0, ErrUnauthorized
   }


   // Start a transaction
   tx, err := db.c.Begin()
   if err != nil {
       return "", "", 0, fmt.Errorf("error starting transaction: %w", err)
   }
  
   // Ensure transaction is rolled back if an error occurs
   defer func() {
       if tx != nil {
           if rollbackErr := tx.Rollback(); rollbackErr != nil {
               logrus.WithError(rollbackErr).Error("Error rolling back transaction")
           }
       }
   }()


   // Get the old group name
   err = tx.QueryRow("SELECT title FROM conversations WHERE id = ? AND is_group = 1", groupID).Scan(&oldName)
   if err != nil {
       if errors.Is(err, sql.ErrNoRows) {
           return "", "", 0, ErrGroupNotFound
       }
       return "", "", 0, fmt.Errorf("error getting old group name: %w", err)
   }
  
   // Check if the new name is the same as the old name
   if oldName == newName {
       // No need to update, just get the member count and return
       err = tx.QueryRow("SELECT COUNT(*) FROM user_conversations WHERE conversation_id = ?", groupID).Scan(&memberCount)
       if err != nil {
           return "", "", 0, fmt.Errorf("error getting member count: %w", err)
       }
      
       // Set tx to nil to prevent rollback in defer function
       tx = nil
      
       return oldName, newName, memberCount, nil
   }
  
   // Check if another group with the same name already exists
   var nameExists int
   err = tx.QueryRow("SELECT COUNT(*) FROM conversations WHERE title = ? AND is_group = 1 AND id != ?", newName, groupID).Scan(&nameExists)
   if err != nil {
       return "", "", 0, fmt.Errorf("error checking for existing group name: %w", err)
   }
   if nameExists > 0 {
       return "", "", 0, ErrNameAlreadyTaken
   }


   // Update the group name in both tables
   _, err = tx.Exec("UPDATE conversations SET title = ? WHERE id = ? AND is_group = 1", newName, groupID)
   if err != nil {
       return "", "", 0, fmt.Errorf("error updating group name in conversations: %w", err)
   }
  
   _, err = tx.Exec("UPDATE groups SET name = ? WHERE id = ?", newName, groupID)
   if err != nil {
       return "", "", 0, fmt.Errorf("error updating group name in groups: %w", err)
   }
  
   // Get the current member count
   err = tx.QueryRow("SELECT COUNT(*) FROM user_conversations WHERE conversation_id = ?", groupID).Scan(&memberCount)
   if err != nil {
       return "", "", 0, fmt.Errorf("error getting member count: %w", err)
   }


   // Commit the transaction
   if err := tx.Commit(); err != nil {
       return "", "", 0, fmt.Errorf("error committing transaction: %w", err)
   }
  
   // Set tx to nil to prevent rollback in defer function
   tx = nil


   return oldName, newName, memberCount, nil
}

func (db *appdbimpl) GetUserByUsername(username string) (User, error) {
	var user User
	err := db.c.QueryRow("SELECT id, name FROM users WHERE name = ?", username).Scan(&user.ID, &user.Name)
	if err != nil {
		return User{}, err
	}
	return user, nil
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
