package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
	"unicode"
	"errors"
	"io"

	"github.com/gerdalukosiute/WASAText/service/api/reqcontext"
	"github.com/gerdalukosiute/WASAText/service/database"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

// Updated response structures to match API documentation
type ConversationDetailsResponse struct {
    ConversationID string                `json:"conversationId"`
    Title          string                `json:"title"`
    IsGroup        bool                  `json:"isGroup"`
    GroupPhotoID   string                `json:"groupPhotoId,omitempty"`
    CreatedAt      string                `json:"createdAt"`
    Participants   []ParticipantResponse `json:"participants"`
    Messages       []MessageResponse     `json:"messages"`
}

type ParticipantResponse struct {
    Username       string `json:"username"`
    UserID         string `json:"userId"`
    ProfilePhotoID string `json:"profilePhotoId,omitempty"`
}

type MessageResponse struct {
    MessageID       string             `json:"messageId"`
    ParentMessageID string             `json:"parentMessageId,omitempty"`
    Sender          SenderResponse     `json:"sender"`
    Type            string             `json:"type"`
    Content         string             `json:"content"`
    Timestamp       string             `json:"timestamp"`
    Status          string             `json:"status"`
    Reactions       []ReactionResponse `json:"reactions,omitempty"`
}

type SenderResponse struct {
    Username string `json:"username"`
    UserID   string `json:"userId"`
}

type ReactionResponse struct {
    Username    string `json:"username"`
    Interaction string `json:"interaction"`
    Content     string `json:"content"`
    Timestamp   string `json:"timestamp"`
}

// ConversationResponse represents the API response for a conversation summary (Updated)
type ConversationResponse struct {
	ConversationID string `json:"conversationId"`
	Title          string `json:"title"`
	CreatedAt      string `json:"createdAt"`
	ProfilePhotoID *string `json:"profilePhotoId,omitempty"`
	IsGroup        bool   `json:"isGroup"`
	LastMessage    struct {
		Type      string `json:"type"`
		Content   string `json:"content"`
		Timestamp string `json:"timestamp"`
	} `json:"lastMessage"`
}

// Handles retrieving the users conversations
func (rt *_router) handleGetConversations(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	ctx.Logger.WithField("userID", userID).Info("Handling get conversations request")

	// Check if the user exists
	userExists, err := rt.db.UserExists(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to check user existence")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}
	if !userExists {
		sendJSONError(w, "User not found", http.StatusUnauthorized)
		return
	}

	conversations, total, err := rt.db.GetUserConversations(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to get user conversations")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}

	// Convert database.Conversation to ConversationResponse
	conversationResponses := make([]ConversationResponse, len(conversations))
	for i, conv := range conversations {
		// Create the LastMessage struct with proper type conversion
		lastMessage := struct {
			Type      string `json:"type"`
			Content   string `json:"content"`
			Timestamp string `json:"timestamp"`
		}{
			Type:      conv.LastMessage.Type,
			Content:   conv.LastMessage.Content,
			Timestamp: conv.LastMessage.Timestamp.Format(time.RFC3339),
		}

		conversationResponses[i] = ConversationResponse{
			ConversationID: conv.ID,                       
			Title:          conv.Title,
			CreatedAt:      conv.CreatedAt.Format(time.RFC3339), 
			ProfilePhotoID: conv.ProfilePhoto,
			IsGroup:        conv.IsGroup,
			LastMessage:    lastMessage,
		}
	}

	// Create the response object according to API spec
	response := struct {
		Conversations []ConversationResponse `json:"conversations"`
		Total         int                    `json:"total"`
	}{
		Conversations: conversationResponses,
		Total:         total,
	}

	ctx.Logger.WithFields(logrus.Fields{
		"conversationCount": len(conversationResponses),
		"totalCount":        total,
	}).Info("Retrieved user conversations")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		ctx.Logger.WithError(err).Error("Failed to encode JSON response")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}
}

// Handles starting a 1 on 1 or group conversation
func (rt *_router) handleStartConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	ctx.Logger.WithField("userID", userID).Info("Handling start conversation request")

	var req struct {
		Recipients []string `json:"recipients"`
		Title      string   `json:"title"`
		IsGroup    bool     `json:"isGroup"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Error("Failed to decode request body")
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Recipients) == 0 {
		ctx.Logger.Error("No recipients provided")
		sendJSONError(w, "At least one recipient is required", http.StatusBadRequest)
		return
	}

	// For group conversations, title is required
	if req.IsGroup && req.Title == "" {
		ctx.Logger.Error("No title provided for group conversation")
		sendJSONError(w, "Title is required for group conversations", http.StatusBadRequest)
		return
	}

	// Get recipient IDs from usernames
	recipientIDs := make([]string, 0, len(req.Recipients))
	for _, recipientName := range req.Recipients {
		recipientID, err := rt.db.GetUserIDByName(recipientName)
		if err != nil {
			ctx.Logger.WithError(err).WithField("recipient", recipientName).Error("Failed to get recipient")
			sendJSONError(w, fmt.Sprintf("Recipient not found: %s", recipientName), http.StatusBadRequest)
			return
		}
		recipientIDs = append(recipientIDs, recipientID)
	}

	// For 1:1 conversations, use the recipient's name as the title if not provided
	title := req.Title
	if !req.IsGroup && len(req.Recipients) == 1 && title == "" {
		title = req.Recipients[0]
	}

	// Start the conversation
	_, err := rt.db.StartConversation(userID, recipientIDs, title, req.IsGroup)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to start conversation")
		if strings.Contains(err.Error(), "participant with ID") {
			sendJSONError(w, fmt.Sprintf("Invalid participant: %v", err), http.StatusBadRequest)
		} else {
			sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		}
		return
	}

	// Reuse the GetUserConversations function to get the response
	conversations, total, err := rt.db.GetUserConversations(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to get user conversations")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}

	// Convert database.Conversation to ConversationResponse
	conversationResponses := make([]ConversationResponse, len(conversations))
	for i, conv := range conversations {
		// Create the LastMessage struct with proper type conversion
		lastMessage := struct {
			Type      string `json:"type"`
			Content   string `json:"content"`
			Timestamp string `json:"timestamp"`
		}{
			Type:      conv.LastMessage.Type,
			Content:   conv.LastMessage.Content,
			Timestamp: conv.LastMessage.Timestamp.Format(time.RFC3339),
		}

		conversationResponses[i] = ConversationResponse{
			ConversationID: conv.ID,
			Title:          conv.Title,
			CreatedAt:      conv.CreatedAt.Format(time.RFC3339),
			ProfilePhotoID: conv.ProfilePhoto,
			IsGroup:        conv.IsGroup,
			LastMessage:    lastMessage,
		}
	}

	// Use the converted response structure
	response := struct {
		Conversations []ConversationResponse `json:"conversations"`
		Total         int                    `json:"total"`
	}{
		Conversations: conversationResponses,
		Total:         total,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		ctx.Logger.WithError(err).Error("Failed to encode response")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}
}

// Handles sending messages
func (rt *_router) handleSendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	conversationID := ps.ByName("conversationId")

	// Check if the user is a participant in the conversation
	isParticipant, err := rt.db.IsUserInConversation(userID, conversationID)
	if err != nil {
		if errors.Is(err, database.ErrConversationNotFound) {
			sendJSONError(w, "Conversation not found", http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("Failed to check user participation in conversation")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}
	if !isParticipant {
		sendJSONError(w, "User is not a participant in this conversation", http.StatusForbidden)
		return
	}

	contentType := r.Header.Get("Content-Type")
	var messageType, content, contentTypeValue string
	var photo []byte
	var parentMessageID *string // Field for parent message ID (for replies)

	// Handle different content types according to API spec
	if strings.HasPrefix(contentType, "application/json") {
		// Handle JSON request for text messages
		var req struct {
			Type           string  `json:"type"`
			Content        string  `json:"content"`
			ParentMessageID *string `json:"parentMessageId,omitempty"` // Optional field for reply
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			ctx.Logger.WithError(err).Error("Failed to decode request body")
			sendJSONError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Type != "text" {
			sendJSONError(w, "Invalid message type for JSON content", http.StatusBadRequest)
			return
		}

		if req.Content == "" {
			sendJSONError(w, "Content is required", http.StatusBadRequest)
			return
		}

		// Check content length
		if len(req.Content) > 1000 {
			sendJSONError(w, "Content exceeds maximum length of 1000 characters", http.StatusRequestEntityTooLarge)
			return
		}

		messageType = req.Type
		content = req.Content
		contentTypeValue = "text/plain"
		parentMessageID = req.ParentMessageID // Store the parent message ID
	} else if strings.HasPrefix(contentType, "multipart/form-data") {
		// Handle multipart form for photo messages
		if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
			ctx.Logger.WithError(err).Error("Failed to parse multipart form")
			sendJSONError(w, "Failed to parse form data", http.StatusBadRequest)
			return
		}

		formType := r.FormValue("type")
		if formType != "photo" {
			sendJSONError(w, "Invalid message type for multipart content", http.StatusBadRequest)
			return
		}

		// Check if this is a reply
		parentMsgValue := r.FormValue("parentMessageId")
		if parentMsgValue != "" {
			parentMessageID = &parentMsgValue
		}

		file, header, err := r.FormFile("photo")
		if err != nil {
			ctx.Logger.WithError(err).Error("Failed to get photo from form")
			sendJSONError(w, "Photo is required", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Check file size (10MB max)
		if header.Size > 10485760 {
			sendJSONError(w, "Photo exceeds maximum size of 10MB", http.StatusRequestEntityTooLarge)
			return
		}

		// Read the file
		photo, err = io.ReadAll(file)
		if err != nil {
			ctx.Logger.WithError(err).Error("Failed to read photo data")
			sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
			return
		}

		if len(photo) < 100 {
			sendJSONError(w, "Photo is too small", http.StatusBadRequest)
			return
		}

		// Detect content type
		contentTypeValue = http.DetectContentType(photo)
		
		// Store the photo in the media_files table
		mediaID, err := rt.db.StoreMediaFile(photo, contentTypeValue)
		if err != nil {
			ctx.Logger.WithError(err).Error("Failed to store media file")
			sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
			return
		}
		
		messageType = "photo"
		// Store the URL to the media in the content field
		content = fmt.Sprintf("/media/%s", mediaID)
	} else {
		sendJSONError(w, "Unsupported content type", http.StatusUnsupportedMediaType)
		return
	}

	// Validate parentMessageID if provided
	if parentMessageID != nil && *parentMessageID != "" {
		// Check if the parent message exists and is in the same conversation
		exists, err := rt.db.ValidateParentMessage(*parentMessageID, conversationID)
		if err != nil {
			ctx.Logger.WithError(err).Error("Failed to validate parent message")
			sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
			return
		}
		if !exists {
			sendJSONError(w, "Parent message not found or not in this conversation", http.StatusBadRequest)
			return
		}
	}

	// Add the message to the database with content type and parent message ID
	messageID, err := rt.db.AddMessage(conversationID, userID, messageType, content, contentTypeValue, parentMessageID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to add message")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}

	// Get the sender's name
	senderName, err := rt.db.GetUserNameByID(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to get sender's name")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}

	// Create the response according to the API documentation
	response := struct {
		MessageID      string `json:"messageId"`
		ConversationID string `json:"conversationId"`
		ParentMessageID *string `json:"parentMessageId,omitempty"`
		Sender         struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		} `json:"sender"`
		Content     string `json:"content"`
		ContentType string `json:"contentType"`
		Type        string `json:"type"`
		Timestamp   string `json:"timestamp"`
		Status      string `json:"status"`
	}{
		MessageID:      messageID,
		ConversationID: conversationID,
		ParentMessageID: parentMessageID,
		Sender: struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		}{
			Username: senderName,
			UserID:   userID,
		},
		Content:     content,
		ContentType: contentTypeValue,
		Type:        messageType,
		Timestamp:   time.Now().Format(time.RFC3339),
		Status:      "delivered", // Initial status is always "delivered"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		ctx.Logger.WithError(err).Error("Failed to encode response")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}
}

// Updated request and response structures for message forwarding
type forwardMessageRequest struct {
	TargetConversationID string `json:"targetConversationId"`
}

type forwardMessageResponse struct {
	NewMessageID         string    `json:"newMessageId"`
	OriginalMessageID    string    `json:"originalMessageId"`
	TargetConversationID string    `json:"targetConversationId"`
	OriginalSender       struct {
		Username string `json:"username"`
		UserID   string `json:"userId"`
	} `json:"originalSender"`
	ForwardedBy struct {
		Username string `json:"username"`
		UserID   string `json:"userId"`
	} `json:"forwardedBy"`
	Content            string    `json:"content"`
	Type               string    `json:"type"`
	OriginalTimestamp  string    `json:"originalTimestamp"`
	ForwardedTimestamp string    `json:"forwardedTimestamp"`
}

// Handles message forwarding
func (rt *_router) handleForwardMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	messageID := ps.ByName("messageId")
	
	ctx.Logger.WithFields(logrus.Fields{
		"userID":    userID,
		"messageID": messageID,
	}).Info("Handling forward message request")

	var req forwardMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Error("Failed to decode request body")
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.TargetConversationID == "" {
		ctx.Logger.Error("Missing required fields in request")
		sendJSONError(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Forward the message
	forwardedMessage, err := rt.db.ForwardMessage(messageID, req.TargetConversationID, userID)
	if err != nil {
		var statusCode int
		var errorMessage string
		
		if errors.Is(err, database.ErrMessageNotFound) {
			statusCode = http.StatusNotFound
			errorMessage = "Original message not found"
		} else if errors.Is(err, database.ErrConversationNotFound) {
			statusCode = http.StatusNotFound
			errorMessage = "Target conversation not found"
		} else if errors.Is(err, database.ErrUnauthorized) {
			statusCode = http.StatusForbidden
			errorMessage = "No permission to forward"
		} else {
			statusCode = http.StatusInternalServerError
			errorMessage = ErrInternalServerMsg
		}
		
		ctx.Logger.WithError(err).Error(errorMessage)
		sendJSONError(w, errorMessage, statusCode)
		return
	}

	// Get the forwarder's name
	forwarderName, err := rt.db.GetUserNameByID(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to get forwarder's name")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}

	// Create the response according to the documentation
	response := forwardMessageResponse{
		NewMessageID:         forwardedMessage.ID,
		OriginalMessageID:    messageID,
		TargetConversationID: req.TargetConversationID,
		OriginalSender: struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		}{
			Username: forwardedMessage.OriginalSender.Name,
			UserID:   forwardedMessage.OriginalSender.ID,
		},
		ForwardedBy: struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		}{
			Username: forwarderName,
			UserID:   userID,
		},
		Content:            forwardedMessage.Content,
		Type:               forwardedMessage.Type,
		OriginalTimestamp:  forwardedMessage.OriginalTimestamp.Format(time.RFC3339),
		ForwardedTimestamp: forwardedMessage.Timestamp.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		ctx.Logger.WithError(err).Error("Failed to encode response")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}
}

// Handler for adding emoji reactions to messages
func (rt *_router) handleAddComment(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	messageID := ps.ByName("messageId")

	ctx.Logger.WithFields(logrus.Fields{
		"messageID": messageID,
		"userID":    userID,
	}).Info("Attempting to add emoji reaction to message")

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Error("Failed to decode request body")
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		ctx.Logger.Error("Empty content provided")
		sendJSONError(w, "Content cannot be empty", http.StatusBadRequest)
		return
	}

	// Validate that the content is an emoji
	if !isValidEmoji(req.Content) {
		ctx.Logger.WithField("content", req.Content).Error("Invalid emoji provided")
		sendJSONError(w, "Content must be a valid emoji", http.StatusBadRequest)
		return
	}

	// Add the emoji reaction
	comment, err := rt.db.AddComment(messageID, userID, req.Content)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to add emoji reaction")
		
		if errors.Is(err, database.ErrUnauthorized) {
			sendJSONError(w, "Unauthorized to add reaction to this message", http.StatusUnauthorized)
			return
		} else if errors.Is(err, database.ErrMessageNotFound) {
			sendJSONError(w, "Message not found", http.StatusNotFound)
			return
		} else {
			sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
			return
		}
	}

	// Get the username for the response
	username, err := rt.db.GetUserNameByID(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to get username")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}

	ctx.Logger.WithFields(logrus.Fields{
		"interactionId": comment.ID,
		"messageID": comment.MessageID,
		"userID":    comment.UserID,
		"content":   comment.Content,
	}).Info("Emoji reaction added successfully")

	// Create the response according to the documentation
	response := struct {
		InteractionID string `json:"interactionId"`
		MessageID     string `json:"messageId"`
		User          struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		} `json:"user"`
		Content   string `json:"content"`
		Timestamp string `json:"timestamp"`
	}{
		InteractionID: comment.ID,
		MessageID:     comment.MessageID,
		User: struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		}{
			Username: username,
			UserID:   comment.UserID,
		},
		Content:   comment.Content,
		Timestamp: comment.Timestamp.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		ctx.Logger.WithError(err).Error("Failed to encode response")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}
}

// isValidEmoji checks if the provided string is a valid emoji
func isValidEmoji(s string) bool {
	// Simple validation for common emoji patterns
	
	// Check if the string is too long to be an emoji
	if utf8.RuneCountInString(s) > 8 {
		return false
	}
	
	// Check if the string contains any ASCII characters (which are not emojis)
	for _, r := range s {
		if r < 128 && !unicode.IsSpace(r) {
			return false
		}
	}
	
	// Check if the string contains at least one emoji-like character
	hasEmojiChar := false
	for _, r := range s {
		// Emoji ranges (this is a simplified check)
		if (r >= 0x1F300 && r <= 0x1F6FF) || // Miscellaneous Symbols and Pictographs
			(r >= 0x2600 && r <= 0x26FF) || // Miscellaneous Symbols
			(r >= 0x2700 && r <= 0x27BF) || // Dingbats
			(r >= 0x1F900 && r <= 0x1F9FF) || // Supplemental Symbols and Pictographs
			(r >= 0x1FA70 && r <= 0x1FAFF) { // Symbols and Pictographs Extended-A
			hasEmojiChar = true
			break
		}
	}
	
	return hasEmojiChar
}

// Handles the request to remove an emoji reaction from a message
func (rt *_router) handleDeleteComment(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	messageID := ps.ByName("messageId")
	commentID := ps.ByName("commentId")

	ctx.Logger.WithFields(logrus.Fields{
		"messageID": messageID,
		"commentID": commentID,
		"userID":    userID,
	}).Info("Attempting to delete emoji reaction")

	err := rt.db.DeleteComment(messageID, commentID, userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to delete emoji reaction")
		w.Header().Set("Content-Type", "application/json")
		
		var statusCode int
		var errorMessage string
		
		if errors.Is(err, database.ErrMessageNotFound) {
			statusCode = http.StatusNotFound
			errorMessage = "Item not found"
		} else if errors.Is(err, database.ErrUnauthorized) {
			statusCode = http.StatusForbidden
			errorMessage = "No permission to remove"
		} else {
			statusCode = http.StatusInternalServerError
			errorMessage = ErrInternalServerMsg
		}
		
		w.WriteHeader(statusCode)
		if encodeErr := json.NewEncoder(w).Encode(map[string]string{"error": errorMessage}); encodeErr != nil {
			ctx.Logger.WithError(encodeErr).Error("Failed to encode error response")
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
		}
		return
	}

	// Get the username of the user who deleted the comment
	username, err := rt.db.GetUserNameByID(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to get username")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if encodeErr := json.NewEncoder(w).Encode(map[string]string{"error": ErrInternalServerMsg}); encodeErr != nil {
			ctx.Logger.WithError(encodeErr).Error("Failed to encode error response")
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
		}
		return
	}

	// Get current time for removedAt field
	removedAt := time.Now().Format(time.RFC3339)

	ctx.Logger.WithFields(logrus.Fields{
		"messageID": messageID,
		"commentID": commentID,
		"userID":    userID,
		"username":  username,
		"removedAt": removedAt,
	}).Info("Emoji reaction deleted successfully")

	// Create response according to the API documentation
	response := struct {
		MessageID    string `json:"messageId"`
		InteractionID string `json:"interactionId"`
		User         struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		} `json:"user"`
		RemovedAt string `json:"removedAt"`
	}{
		MessageID:    messageID,
		InteractionID: commentID,
		User: struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		}{
			Username: username,
			UserID:   userID,
		},
		RemovedAt: removedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		ctx.Logger.WithError(encodeErr).Error("Failed to encode success response")
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// Handles status updates
func (rt *_router) handleUpdateMessageStatus(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	messageID := ps.ByName("messageId")

	ctx.Logger.WithFields(logrus.Fields{
		"messageID": messageID,
		"userID":    userID,
	}).Info("Handling update message status request")

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Error("Invalid request body")
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the status
	validStatuses := []string{"delivered", "read"}
	isValid := false
	for _, status := range validStatuses {
		if req.Status == status {
			isValid = true
			break
		}
	}
	if !isValid {
		ctx.Logger.WithField("status", req.Status).Error("Invalid status")
		sendJSONError(w, "Invalid status", http.StatusBadRequest)
		return
	}

	// Update the message status
	statusUpdate, err := rt.db.UpdateMessageStatus(messageID, userID, req.Status)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to update message status")
		
		var statusCode int
		var errorMessage string
		
		if errors.Is(err, database.ErrMessageNotFound) {
			statusCode = http.StatusNotFound
			errorMessage = "Message not found"
		} else if errors.Is(err, database.ErrUnauthorized) {
			statusCode = http.StatusForbidden
			errorMessage = "Not permitted"
		} else {
			statusCode = http.StatusInternalServerError
			errorMessage = ErrInternalServerMsg
		}
		
		sendJSONError(w, errorMessage, statusCode)
		return
	}

	// Create the response according to the API documentation
	response := struct {
		MessageID      string    `json:"messageId"`
		Status         string    `json:"status"`
		UpdatedBy      struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		} `json:"updatedBy"`
		UpdatedAt      string    `json:"updatedAt"`
		ConversationID string    `json:"conversationId"`
	}{
		MessageID: statusUpdate.MessageID,
		Status:    statusUpdate.Status,
		UpdatedBy: struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		}{
			Username: statusUpdate.UpdatedBy.Name,
			UserID:   statusUpdate.UpdatedBy.ID,
		},
		UpdatedAt:      statusUpdate.UpdatedAt.Format(time.RFC3339),
		ConversationID: statusUpdate.ConversationID,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		ctx.Logger.WithError(err).Error("Failed to encode response")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}
}

// Handles message deletion
func (rt *_router) handleDeleteMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	ctx.Logger.WithFields(logrus.Fields{
		"userID":    userID,
		"messageID": ps.ByName("messageId"),
	}).Info("Handling delete message request")

	messageID := ps.ByName("messageId")

	// Delete the message
	deletedMessage, conversationID, err := rt.db.DeleteMessage(messageID, userID)
	if err != nil {
		var statusCode int
		var errorMessage string
		
		if errors.Is(err, database.ErrMessageNotFound) {
			statusCode = http.StatusNotFound
			errorMessage = "Message not found"
		} else if errors.Is(err, database.ErrUnauthorized) {
			statusCode = http.StatusForbidden
			errorMessage = "No permission to delete"
		} else {
			statusCode = http.StatusInternalServerError
			errorMessage = ErrInternalServerMsg
			ctx.Logger.WithError(err).Error("Failed to delete message")
		}
		
		sendJSONError(w, errorMessage, statusCode)
		return
	}

	// Get the username for the response
	username, err := rt.db.GetUserNameByID(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to get username")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}

	// Create the response according to the API documentation
	response := struct {
		MessageID      string    `json:"messageId"`
		User           struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		} `json:"user"`
		DeletedAt      string    `json:"deletedAt"`
		ConversationID string    `json:"conversationId"`
	}{
		MessageID: deletedMessage.ID,
		User: struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		}{
			Username: username,
			UserID:   userID,
		},
		DeletedAt:      time.Now().Format(time.RFC3339),
		ConversationID: conversationID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		ctx.Logger.WithError(err).Error("Failed to encode response")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}
}

// Convert database messages to response format
func convertMessages(dbMessages []database.Message) []MessageResponse {
	messages := make([]MessageResponse, len(dbMessages))
	for i, m := range dbMessages {
		messages[i] = MessageResponse{
			MessageID: m.ID,
			Sender: SenderResponse{
				Username: m.Sender,
				UserID:   m.SenderID,
			},
			Type:      m.Type,
			Content:   m.Content,
			Timestamp: m.Timestamp.Format(time.RFC3339),
			Status:    m.Status,
			Reactions: convertReactions(m.Comments),
		}
		
		// Add parent message ID if present
		if m.ParentMessageID != nil {
			messages[i].ParentMessageID = *m.ParentMessageID
		}
	}
	return messages
}

// Convert database comments to reaction responses
func convertReactions(dbComments []database.Comment) []ReactionResponse {
	reactions := make([]ReactionResponse, len(dbComments))
	for i, c := range dbComments {
		reactions[i] = ReactionResponse{
			Username:    c.Username,
			Interaction: "reaction",
			Content:     c.Content,
			Timestamp:   c.Timestamp.Format(time.RFC3339),
		}
	}
	return reactions
}

// Convert database participants to response format
func convertParticipants(dbParticipants []database.Participant) []ParticipantResponse {
	participants := make([]ParticipantResponse, len(dbParticipants))
	for i, p := range dbParticipants {
		participants[i] = ParticipantResponse{
			Username:       p.Name,
			UserID:         p.ID,
			ProfilePhotoID: p.PhotoID,
		}
	}
	return participants
}

// Handler for getting conversation details
func (rt *_router) handleGetConversationDetails(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	ctx.Logger.WithField("userID", userID).Info("Handling get conversation details request")

	conversationID := ps.ByName("conversationId")
	if conversationID == "" {
		ctx.Logger.Error("Missing conversationId in request")
		sendJSONError(w, "ConversationId is required", http.StatusBadRequest)
		return
	}

	conversation, err := rt.db.GetConversationDetails(conversationID, userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to get conversation details")
		
		if errors.Is(err, database.ErrConversationNotFound) {
			sendJSONError(w, "Conversation not found", http.StatusNotFound)
			return
		}
		
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}

	// Create the response according to the API documentation
	response := ConversationDetailsResponse{
		ConversationID: conversation.ID,
		Title:          conversation.Title,
		IsGroup:        conversation.IsGroup,
		CreatedAt:      conversation.CreatedAt.Format(time.RFC3339),
		Participants:   convertParticipants(conversation.Participants),
		Messages:       convertMessages(conversation.Messages),
	}
	
	// Add group photo ID if present and it's a group
	if conversation.IsGroup && conversation.ProfilePhoto != "" {
		response.GroupPhotoID = conversation.ProfilePhoto
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		ctx.Logger.WithError(err).Error("Failed to encode response")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}
}