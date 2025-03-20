package database

import (
	"database/sql"
	"fmt"
	"time"
	"unicode/utf8"
	"errors"
	"math/rand"
	"log"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// Database operation to retrieve user conversations
func (db *appdbimpl) GetUserConversations(userID string) ([]Conversation, int, error) {
	logrus.WithField("userID", userID).Info("Getting user conversations")
	
	// First, get the total count of conversations
	countQuery := `
	SELECT COUNT(DISTINCT c.id)
	FROM users u
	JOIN user_conversations uc ON u.id = uc.user_id
	JOIN conversations c ON uc.conversation_id = c.id
	WHERE u.id = ?
	`
	
	var total int
	err := db.c.QueryRow(countQuery, userID).Scan(&total)
	if err != nil {
		logrus.WithError(err).Error("Error counting user conversations")
		return nil, 0, fmt.Errorf("error counting user conversations: %w", err)
	}
	
	// Now get the conversations with details
	query := `
	SELECT c.id, c.title, c.is_group, c.created_at,
		   CASE
			   WHEN c.is_group = 0 THEN (
				   SELECT u.name
				   FROM users u
				   JOIN user_conversations uc ON u.id = uc.user_id
				   WHERE uc.conversation_id = c.id AND u.id != ?
			   )
			   ELSE c.title
		   END as display_title,
		   CASE
			   WHEN c.is_group = 0 THEN (
				   SELECT u.photo_id
				   FROM users u
				   JOIN user_conversations uc ON u.id = uc.user_id
				   WHERE uc.conversation_id = c.id AND u.id != ?
			   )
			   ELSE c.profile_photo
		   END as display_photo,
		   m.type, m.content, m.created_at as message_timestamp
	FROM users u
	JOIN user_conversations uc ON u.id = uc.user_id
	JOIN conversations c ON uc.conversation_id = c.id
	LEFT JOIN (
		SELECT m1.*
		FROM messages m1
		INNER JOIN (
			SELECT conversation_id, MAX(created_at) as max_created_at
			FROM messages
			GROUP BY conversation_id
		) m2 ON m1.conversation_id = m2.conversation_id AND m1.created_at = m2.max_created_at
	) m ON c.id = m.conversation_id
	WHERE u.id = ?
	ORDER BY COALESCE(m.created_at, c.created_at) DESC
	LIMIT 10000
	`

	rows, err := db.c.Query(query, userID, userID, userID)
	if err != nil {
		logrus.WithError(err).Error("Error querying user conversations")
		return nil, 0, fmt.Errorf("error querying user conversations: %w", err)
	}
	defer rows.Close()

	var conversations []Conversation
    for rows.Next() {
        var conv Conversation
        var displayTitle, displayPhoto, messageType, messageContent sql.NullString
        var messageTimestamp, conversationCreatedAt sql.NullTime

        err := rows.Scan(
            &conv.ID,
            &conv.Title,
            &conv.IsGroup,
            &conversationCreatedAt,
            &displayTitle,
            &displayPhoto,
            &messageType,
            &messageContent,
            &messageTimestamp,
        )
        if err != nil {
            logrus.WithError(err).Error("Error scanning conversation row")
            return nil, 0, fmt.Errorf("error scanning conversation row: %w", err)
        }

        // Use the display title from the query
        if displayTitle.Valid {
            conv.Title = displayTitle.String
        }
        
        // Set the profile photo
        if displayPhoto.Valid {
            conv.ProfilePhoto = &displayPhoto.String
        }
        
        // Set the creation time
        if conversationCreatedAt.Valid {
            conv.CreatedAt = conversationCreatedAt.Time
        }

        // Set the last message details
        var msgType, msgContent string
        var msgTimestamp time.Time
        
        if messageType.Valid {
            msgType = messageType.String
        } else {
            msgType = ""
        }
        
        if messageContent.Valid {
            msgContent = messageContent.String
        } else {
            msgContent = ""
        }
        
        if messageTimestamp.Valid {
            msgTimestamp = messageTimestamp.Time
        } else {
            msgTimestamp = time.Time{} // Zero value for time.Time
        }
        
        conv.LastMessage = struct {
            Type      string
            Content   string
            Timestamp time.Time
        }{
            Type:      msgType,
            Content:   msgContent,
            Timestamp: msgTimestamp,
        }

        conversations = append(conversations, conv)
    }

	if err := rows.Err(); err != nil {
		logrus.WithError(err).Error("Error iterating conversation rows")
		return nil, 0, fmt.Errorf("error iterating conversation rows: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"userID":            userID,
		"conversationCount": len(conversations),
		"totalCount":        total,
	}).Info("Retrieved user conversations")

	return conversations, total, nil
}

func (db *appdbimpl) StartConversation(initiatorID string, recipientIDs []string, title string, isGroup bool) (string, error) {
	tx, err := db.c.Begin()
	if err != nil {
		return "", fmt.Errorf("error starting transaction: %w", err)
	}
	
	// Defer rollback - will be a no-op if transaction is committed
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			log.Printf("Error rolling back transaction: %v", rollbackErr)
		}
	}()

	// For 1:1 conversations, check if a conversation already exists
	if !isGroup && len(recipientIDs) == 1 {
		existingID, exists, err := db.GetExistingConversation(initiatorID, recipientIDs[0])
		if err != nil {
			return "", fmt.Errorf("error checking for existing conversation: %w", err)
		}
		if exists {
			// If a conversation already exists, commit the empty transaction and return the existing ID
			if err := tx.Commit(); err != nil {
				return "", fmt.Errorf("error committing transaction: %w", err)
			}
			return existingID, nil
		}
	}

	// Generate a conversation ID that matches the pattern ^[a-zA-Z0-9_-]{6,20}$
	conversationID, err := db.GenerateConversationID()
	if err != nil {
		return "", fmt.Errorf("error generating conversation ID: %w", err)
	}

	// Current time for created_at
	now := time.Now()

	// Insert the new conversation
	_, err = tx.Exec("INSERT INTO conversations (id, title, profile_photo, is_group, created_at) VALUES (?, ?, NULL, ?, ?)",
		conversationID, title, isGroup, now)
	if err != nil {
		return "", fmt.Errorf("error creating conversation: %w", err)
	}

	// If it's a group, also insert into the groups table
	if isGroup {
		_, err = tx.Exec("INSERT INTO groups (id, name) VALUES (?, ?)", conversationID, title)
		if err != nil {
			return "", fmt.Errorf("error creating group: %w", err)
		}
	}

	// Add all participants (including the initiator) to the conversation
	participants := append([]string{initiatorID}, recipientIDs...)
	
	// Remove duplicates from participants
	uniqueParticipants := make([]string, 0, len(participants))
	seen := make(map[string]bool)
	for _, p := range participants {
		if !seen[p] {
			seen[p] = true
			uniqueParticipants = append(uniqueParticipants, p)
		}
	}
	
	for _, participantID := range uniqueParticipants {
		// Check if the participant exists
		var exists bool
		err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", participantID).Scan(&exists)
		if err != nil {
			return "", fmt.Errorf("error checking participant existence: %w", err)
		}
		if !exists {
			return "", fmt.Errorf("participant with ID %s does not exist", participantID)
		}

		// Add participant to the conversation
		_, err = tx.Exec("INSERT INTO user_conversations (user_id, conversation_id) VALUES (?, ?)",
			participantID, conversationID)
		if err != nil {
			return "", fmt.Errorf("error adding participant %s to conversation: %w", participantID, err)
		}

		// If it's a group, also add to group_members
		if isGroup {
			_, err = tx.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?, ?)",
				conversationID, participantID)
			if err != nil {
				return "", fmt.Errorf("error adding participant %s to group: %w", participantID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("error committing transaction: %w", err)
	}

	return conversationID, nil
}

func (db *appdbimpl) GetExistingConversation(userID1, userID2 string) (string, bool, error) {
    // Find conversations where both users are participants and it's not a group
    query := `
        SELECT c.id 
        FROM conversations c
        JOIN user_conversations uc1 ON c.id = uc1.conversation_id
        JOIN user_conversations uc2 ON c.id = uc2.conversation_id
        WHERE c.is_group = 0
        AND uc1.user_id = ?
        AND uc2.user_id = ?
    `
    
    var conversationID string
    err := db.c.QueryRow(query, userID1, userID2).Scan(&conversationID)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return "", false, nil
        }
        return "", false, fmt.Errorf("error checking for existing conversation: %w", err)
    }
    
    return conversationID, true, nil
}

// Creates a unique conversation ID that matches the pattern ^[a-zA-Z0-9_-]{6,20}$
func (db *appdbimpl) GenerateConversationID() (string, error) {
    // Try up to 10 times to generate a unique ID
    for i := 0; i < 10; i++ {
        // Generate a random number between 100 and 999999
        // This will result in IDs between 7 and 10 characters long ("chat" + 3-6 digits)
        randomNum := 100 + rand.Intn(999900)
        candidateID := fmt.Sprintf("chat%d", randomNum)
        
        // Check if this ID already exists
        var exists bool
        err := db.c.QueryRow("SELECT EXISTS(SELECT 1 FROM conversations WHERE id = ?)", candidateID).Scan(&exists)
        if err != nil {
            return "", fmt.Errorf("error checking conversation ID existence: %w", err)
        }
        
        // If the ID doesn't exist, return it
        if !exists {
            return candidateID, nil
        }
    }
    
    // If it couldn't generate a unique ID after 10 attempts, return an error
    return "", fmt.Errorf("failed to generate a unique conversation ID after multiple attempts")
}

// UPDATED TO THIS POINT 

// AddMessage adds a new message to a conversation and returns the message ID
func (db *appdbimpl) AddMessage(conversationID, senderID, messageType, content string) (string, error) {
	messageID := uuid.Must(uuid.NewV4()).String()
	_, err := db.c.Exec(`
		INSERT INTO messages (id, conversation_id, sender_id, type, content, created_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, messageID, conversationID, senderID, messageType, content, time.Now(), "delivered")

	if err != nil {
		return "", fmt.Errorf("error adding message: %w", err)
	}

	return messageID, nil
}

func (db *appdbimpl) ForwardMessage(originalMessageID, targetConversationID, userID string) (*Message, error) {
	// Check if the original message exists
	var originalMessageExists bool
	err := db.c.QueryRow("SELECT EXISTS(SELECT 1 FROM messages WHERE id = ?)", originalMessageID).Scan(&originalMessageExists)
	if err != nil {
		return nil, fmt.Errorf("error checking message existence: %w", err)
	}
	if !originalMessageExists {
		return nil, ErrMessageNotFound
	}

	// Check if the user is part of the original conversation
	isAuthorized, err := db.isUserAuthorized(userID, originalMessageID)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, ErrUnauthorized
	}

	// Check if the target conversation exists
	exists, err := db.conversationExists(targetConversationID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrConversationNotFound
	}

	// Start a transaction
	tx, err := db.c.Begin()
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// Fetch the original message
	var originalMessage Message
	err = tx.QueryRow(`
        SELECT m.id, m.sender_id, u.name, m.type, m.content, m.created_at, m.status
        FROM messages m
        JOIN users u ON m.sender_id = u.id
        WHERE m.id = ?
    `, originalMessageID).Scan(
		&originalMessage.ID,
		&originalMessage.SenderID,
		&originalMessage.Sender,
		&originalMessage.Type,
		&originalMessage.Content,
		&originalMessage.Timestamp,
		&originalMessage.Status,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMessageNotFound
		}
		return nil, fmt.Errorf("error fetching original message: %w", err)
	}

	// Check if the user is part of the target conversation
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM user_conversations WHERE user_id = ? AND conversation_id = ?", userID, targetConversationID).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("error checking user participation: %w", err)
	}
	if count == 0 {
		return nil, ErrUnauthorized
	}

	// Create the new forwarded message
	newMessageID := uuid.Must(uuid.NewV4()).String()
	_, err = tx.Exec(`
        INSERT INTO messages (id, conversation_id, sender_id, type, content, created_at, status)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, newMessageID, targetConversationID, userID, originalMessage.Type, originalMessage.Content, time.Now(), "sent")
	if err != nil {
		return nil, fmt.Errorf("error inserting forwarded message: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	// Fetch the newly created message
	var newMessage Message
	err = db.c.QueryRow(`
        SELECT m.id, m.sender_id, u.name, m.type, m.content, m.created_at, m.status
        FROM messages m
        JOIN users u ON m.sender_id = u.id
        WHERE m.id = ?
    `, newMessageID).Scan(
		&newMessage.ID,
		&newMessage.SenderID,
		&newMessage.Sender,
		&newMessage.Type,
		&newMessage.Content,
		&newMessage.Timestamp,
		&newMessage.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("error fetching new message: %w", err)
	}

	return &newMessage, nil
}

func (db *appdbimpl) conversationExists(conversationID string) (bool, error) {
	var count int
	err := db.c.QueryRow("SELECT COUNT(*) FROM conversations WHERE id = ?", conversationID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *appdbimpl) DeleteMessage(messageID, userID string) (*Message, error) {
	var messageToDelete Message
	var icon sql.NullString

	// Find the message and check if the user is authorized to delete it
	err := db.c.QueryRow(`
		SELECT id, type, content, icon, sender_id, created_at, status
		FROM messages 
		WHERE id = ?`, messageID).Scan(
		&messageToDelete.ID,
		&messageToDelete.Type,
		&messageToDelete.Content,
		&icon,
		&messageToDelete.Sender,
		&messageToDelete.Timestamp,
		&messageToDelete.Status,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMessageNotFound
		}
		return nil, fmt.Errorf("error querying message: %w", err)
	}

	// Handle NULL icon
	if icon.Valid {
		messageToDelete.Icon = icon.String
	} else {
		messageToDelete.Icon = ""
	}

	if messageToDelete.Sender != userID {
		return nil, ErrUnauthorized
	}

	// Start a transaction
	tx, err := db.c.Begin()
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback() // Rollback the transaction if it's not committed

	// Delete associated reactions
	_, err = tx.Exec("DELETE FROM comments WHERE message_id = ?", messageID)
	if err != nil {
		return nil, fmt.Errorf("error deleting reactions: %w", err)
	}

	// Delete the message
	result, err := tx.Exec("DELETE FROM messages WHERE id = ?", messageID)
	if err != nil {
		return nil, fmt.Errorf("error deleting message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, ErrMessageNotFound
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return &messageToDelete, nil
}

func (db *appdbimpl) UpdateMessageStatus(messageID, userID, newStatus string) error {
	// Start a transaction
	tx, err := db.c.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if the user is authorized to update this message
	var conversationID string
	var currentStatus string
	err = tx.QueryRow("SELECT conversation_id, status FROM messages WHERE id = ?", messageID).Scan(&conversationID, &currentStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrMessageNotFound
		}
		return fmt.Errorf("error fetching message: %w", err)
	}

	// Check if the user is part of the conversation
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM user_conversations WHERE user_id = ? AND conversation_id = ?", userID, conversationID).Scan(&count)
	if err != nil {
		return fmt.Errorf("error checking user authorization: %w", err)
	}
	if count == 0 {
		return ErrUnauthorized
	}

	// Check if it's a group conversation
	var participantCount int
	err = tx.QueryRow("SELECT COUNT(*) FROM user_conversations WHERE conversation_id = ?", conversationID).Scan(&participantCount)
	if err != nil {
		return fmt.Errorf("error checking conversation type: %w", err)
	}

	if participantCount > 2 {
		// It's a group conversation
		// Update or insert the user's read status
		_, err = tx.Exec("INSERT INTO message_read_status (message_id, user_id, status) VALUES (?, ?, ?) ON CONFLICT(message_id, user_id) DO UPDATE SET status = ?", messageID, userID, newStatus, newStatus)
		if err != nil {
			return fmt.Errorf("error updating user read status: %w", err)
		}

		// Check if all participants (except the sender) have read the message
		var readCount int
		err = tx.QueryRow("SELECT COUNT(*) FROM message_read_status WHERE message_id = ? AND status = 'read'", messageID).Scan(&readCount)
		if err != nil {
			return fmt.Errorf("error checking read status: %w", err)
		}

		if readCount == participantCount-1 { // All participants except the sender have read the message
			newStatus = "read"
		} else {
			newStatus = "delivered"
		}
	}

	// Update the message status if it's changing
	if currentStatus != newStatus {
		_, err = tx.Exec("UPDATE messages SET status = ? WHERE id = ?", newStatus, messageID)
		if err != nil {
			return fmt.Errorf("error updating message status: %w", err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

func (db *appdbimpl) GetMessageByID(messageID string) (*Message, error) {
	query := `
        SELECT m.id, m.sender_id, u.name AS sender, m.type, m.content, m.icon, m.created_at, m.status
        FROM messages m
        JOIN users u ON m.sender_id = u.id
        WHERE m.id = ?
    `
	var msg Message
	var icon sql.NullString // Use sql.NullString to handle potential NULL values
	err := db.c.QueryRow(query, messageID).Scan(
		&msg.ID, &msg.SenderID, &msg.Sender, &msg.Type, &msg.Content, &icon, &msg.Timestamp, &msg.Status,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMessageNotFound
		}
		return nil, fmt.Errorf("error fetching message: %w", err)
	}

	// Set the Icon field based on the sql.NullString value
	if icon.Valid {
		msg.Icon = icon.String
	} else {
		msg.Icon = "" // or set a default value if preferred
	}

	// Fetch comments for the message
	commentsQuery := `
        SELECT c.id, c.message_id, c.user_id, u.name AS username, c.content, c.created_at
        FROM comments c
        JOIN users u ON c.user_id = u.id
        WHERE c.message_id = ?
        ORDER BY c.created_at ASC
    `
	rows, err := db.c.Query(commentsQuery, messageID)
	if err != nil {
		return nil, fmt.Errorf("error fetching comments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.ID, &comment.MessageID, &comment.UserID, &comment.Username, &comment.Content, &comment.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("error scanning comment: %w", err)
		}
		msg.Comments = append(msg.Comments, comment)
	}

	return &msg, nil
}

func (db *appdbimpl) isUserAuthorized(userID string, messageID string) (bool, error) {
	var count int
	err := db.c.QueryRow(`
		SELECT COUNT(*) 
		FROM messages m
		JOIN user_conversations uc ON m.conversation_id = uc.conversation_id
		WHERE m.id = ? AND uc.user_id = ?
	`, messageID, userID).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("error checking user authorization: %w", err)
	}

	return count > 0, nil
}

func (db *appdbimpl) AddComment(messageID, userID, content string) (*Comment, error) {
	// Start a transaction
	tx, err := db.c.Begin()
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback() // Rollback the transaction if it's not committed

	// Check if the message exists
	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM messages WHERE id = ?)", messageID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("error checking message existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("message not found")
	}

	// Check if the user is authorized to comment on this message
	isAuthorized, err := db.isUserAuthorized(userID, messageID)
	if err != nil {
		return nil, fmt.Errorf("error checking user authorization: %w", err)
	}
	if !isAuthorized {
		return nil, fmt.Errorf("user not authorized to comment on this message")
	}

	isEmoji := utf8.RuneCountInString(content) <= 2 // Allow for heart emoji (2 runes)

	var commentID string
	timestamp := time.Now().UTC()

	if isEmoji {
		// Check if the user has already reacted to this message
		var existingCommentID string
		err = tx.QueryRow(`
			SELECT id FROM comments 
			WHERE message_id = ? AND user_id = ? AND LENGTH(content) <= 4
		`, messageID, userID).Scan(&existingCommentID)

		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("error checking existing reaction: %w", err)
		}

		if existingCommentID != "" {
			// Update existing reaction
			_, err = tx.Exec(`
				UPDATE comments 
				SET content = ?, created_at = ?
				WHERE id = ?
			`, content, timestamp, existingCommentID)
			if err != nil {
				return nil, fmt.Errorf("error updating existing reaction: %w", err)
			}
			commentID = existingCommentID
		} else {
			// Insert new reaction
			commentID = uuid.Must(uuid.NewV4()).String()
			_, err = tx.Exec(`
				INSERT INTO comments (id, message_id, user_id, content, created_at)
				VALUES (?, ?, ?, ?, ?)
			`, commentID, messageID, userID, content, timestamp)
			if err != nil {
				return nil, fmt.Errorf("error inserting new reaction: %w", err)
			}
		}
	} else {
		// Insert the new comment
		commentID = uuid.Must(uuid.NewV4()).String()
		_, err = tx.Exec(`
			INSERT INTO comments (id, message_id, user_id, content, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, commentID, messageID, userID, content, timestamp)
		if err != nil {
			return nil, fmt.Errorf("error inserting comment: %w", err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return &Comment{
		ID:        commentID,
		MessageID: messageID,
		UserID:    userID,
		Content:   content,
		Timestamp: timestamp,
	}, nil
}

func (db *appdbimpl) DeleteComment(messageID, commentID, userID string) error {
	// Start a transaction
	tx, err := db.c.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback() // Rollback the transaction if it's not committed

	// Check if the user is authorized to access the message
	isAuthorized, err := db.isUserAuthorized(userID, messageID)
	if err != nil {
		return fmt.Errorf("error checking user authorization: %w", err)
	}
	if !isAuthorized {
		return fmt.Errorf("user not authorized to access this message")
	}

	// Check if the comment exists and get its user ID
	var commentUserID string
	err = tx.QueryRow("SELECT user_id FROM comments WHERE id = ? AND message_id = ?", commentID, messageID).Scan(&commentUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("comment not found")
		}
		return fmt.Errorf("error checking comment: %w", err)
	}

	// Check if the user is the owner of the comment
	if commentUserID != userID {
		return fmt.Errorf("user not authorized to delete this comment")
	}

	// Delete the comment
	result, err := tx.Exec("DELETE FROM comments WHERE id = ?", commentID)
	if err != nil {
		return fmt.Errorf("error deleting comment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("comment not found")
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

func (db *appdbimpl) GetConversationDetails(conversationID, userID string) (*ConversationDetails, error) {
	// First, check if the user is a participant in the conversation
	var count int
	err := db.c.QueryRow("SELECT COUNT(*) FROM user_conversations WHERE conversation_id = ? AND user_id = ?", conversationID, userID).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("error checking user participation: %w", err)
	}
	if count == 0 {
		return nil, ErrConversationNotFound
	}

	// Get conversation details
	var details ConversationDetails
	err = db.c.QueryRow("SELECT id, title, is_group, updated_at FROM conversations WHERE id = ?", conversationID).Scan(
		&details.ID,
		&details.Title,
		&details.IsGroup,
		&details.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrConversationNotFound
		}
		return nil, fmt.Errorf("error fetching conversation details: %w", err)
	}

	// Get participants
	rows, err := db.c.Query(`
        SELECT u.id, u.name
        FROM users u
        JOIN user_conversations uc ON u.id = uc.user_id
        WHERE uc.conversation_id = ?
    `, conversationID)
	if err != nil {
		return nil, fmt.Errorf("error fetching participants: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var participant Participant
		if err := rows.Scan(&participant.ID, &participant.Name); err != nil {
			return nil, fmt.Errorf("error scanning participant: %w", err)
		}
		details.Participants = append(details.Participants, participant)
	}

	// Get messages
	rows, err = db.c.Query(`
        SELECT m.id, u.id, u.name, m.type, m.content, m.icon, m.created_at, m.status
        FROM messages m
        JOIN users u ON m.sender_id = u.id
        WHERE m.conversation_id = ?
        ORDER BY m.created_at DESC
    `, conversationID)
	if err != nil {
		return nil, fmt.Errorf("error fetching messages: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var msg Message
		var icon sql.NullString // Use sql.NullString for the icon field
		if err := rows.Scan(&msg.ID, &msg.SenderID, &msg.Sender, &msg.Type, &msg.Content, &icon, &msg.Timestamp, &msg.Status); err != nil {
			return nil, fmt.Errorf("error scanning message: %w", err)
		}

		// Handle the NULL case for icon
		if icon.Valid {
			msg.Icon = icon.String
		} else {
			msg.Icon = "" // or some default value
		}

		// Fetch comments for this message
		comments, err := db.GetComments(msg.ID)
		if err != nil {
			return nil, fmt.Errorf("error fetching comments: %w", err)
		}
		msg.Comments = comments

		details.Messages = append(details.Messages, msg)
	}

	return &details, nil
}

func (db *appdbimpl) GetComments(messageID string) ([]Comment, error) {
	rows, err := db.c.Query(`
		SELECT c.id, c.message_id, c.user_id, u.name, c.content, c.created_at
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.message_id = ?
		ORDER BY c.created_at
	`, messageID)
	if err != nil {
		return nil, fmt.Errorf("error fetching comments: %w", err)
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.MessageID, &c.UserID, &c.Username, &c.Content, &c.Timestamp); err != nil {
			return nil, fmt.Errorf("error scanning comment: %w", err)
		}
		comments = append(comments, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating comments: %w", err)
	}

	return comments, nil
}

func (db *appdbimpl) IsUserInConversation(userID, conversationID string) (bool, error) {
	var exists bool
	err := db.c.QueryRow("SELECT EXISTS(SELECT 1 FROM user_conversations WHERE user_id = ? AND conversation_id = ?)", userID, conversationID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, ErrConversationNotFound
		}
		return false, fmt.Errorf("error checking user participation: %w", err)
	}
	return exists, nil
}
