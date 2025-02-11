package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gerdalukosiute/WASAText/service/api/reqcontext"
	"github.com/gerdalukosiute/WASAText/service/database"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

// GroupResponse represents the API response for a group
type GroupResponse struct {
	ID   string `json:"groupId"`
	Name string `json:"groupName"`
}

// ConversationDetailsResponse represents the API response for conversation details
type ConversationDetailsResponse struct {
	ID           string                `json:"id"`
	Title        string                `json:"title"`
	IsGroup      bool                  `json:"isGroup"`
	UpdatedAt    time.Time             `json:"updatedAt"`
	Participants []ParticipantResponse `json:"participants"`
	Messages     []MessageResponse     `json:"messages"`
}

// ParticipantResponse represents a participant in the API response
type ParticipantResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// MessageResponse represents a message in the API response
type MessageResponse struct {
	ID        string            `json:"id"`
	SenderID  string            `json:"senderId"`
	Sender    string            `json:"sender"`
	Type      string            `json:"type"`
	Content   string            `json:"content"`
	Icon      string            `json:"icon"`
	Timestamp time.Time         `json:"timestamp"`
	Status    string            `json:"status"`
	Comments  []CommentResponse `json:"comments"`
}

// CommentResponse represents a comment in the API response
type CommentResponse struct {
	ID        string    `json:"id"`
	MessageID string    `json:"messageId"`
	UserID    string    `json:"userId"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ConversationResponse represents the API response for a conversation summary
type ConversationResponse struct {
	ID           string          `json:"conversationId"`
	Title        string          `json:"title"`
	ProfilePhoto *string         `json:"profilePhoto"`
	IsGroup      bool            `json:"isGroup"`
	LastMessage  MessageResponse `json:"lastMessage"`
}

func (rt *_router) handleGetConversations(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	ctx.Logger.WithField("userID", userID).Info("Handling get conversations request")

	// Check if the user exists
	userExists, err := rt.db.UserExists(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to check user existence")
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !userExists {
		sendJSONError(w, "User not found", http.StatusUnauthorized)
		return
	}

	conversations, err := rt.db.GetUserConversations(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to get user conversations")
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]ConversationResponse, len(conversations))
	for i, conv := range conversations {
		response[i] = ConversationResponse{
			ID:           conv.ID,
			Title:        conv.Title,
			ProfilePhoto: conv.ProfilePhoto,
			IsGroup:      conv.IsGroup,
			LastMessage: MessageResponse{
				ID:        conv.LastMessage.ID,
				SenderID:  conv.LastMessage.SenderID,
				Sender:    conv.LastMessage.Sender,
				Type:      conv.LastMessage.Type,
				Content:   conv.LastMessage.Content,
				Icon:      conv.LastMessage.Icon,
				Timestamp: conv.LastMessage.Timestamp,
				Status:    conv.LastMessage.Status,
				Comments:  []CommentResponse{}, // Assuming comments are not fetched in this query
			},
		}
	}

	ctx.Logger.WithField("conversationCount", len(response)).Info("Retrieved user conversations")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func convertMessages(dbMessages []database.Message) []MessageResponse {
	messages := make([]MessageResponse, len(dbMessages))
	for i, m := range dbMessages {
		messages[i] = MessageResponse{
			ID:        m.ID,
			SenderID:  m.SenderID,
			Sender:    m.Sender,
			Type:      m.Type,
			Content:   m.Content,
			Icon:      m.Icon,
			Timestamp: m.Timestamp,
			Status:    m.Status,
			Comments:  convertComments(m.Comments),
		}
	}
	return messages
}

func convertComments(dbComments []database.Comment) []CommentResponse {
	comments := make([]CommentResponse, len(dbComments))
	for i, c := range dbComments {
		comments[i] = CommentResponse{
			ID:        c.ID,
			MessageID: c.MessageID,
			UserID:    c.UserID,
			Username:  c.Username,
			Content:   c.Content,
			Timestamp: c.Timestamp,
		}
	}
	return comments
}

func convertParticipants(dbParticipants []database.Participant) []ParticipantResponse {
	participants := make([]ParticipantResponse, len(dbParticipants))
	for i, p := range dbParticipants {
		participants[i] = ParticipantResponse{
			ID:   p.ID,
			Name: p.Name,
		}
	}
	return participants
}

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
		if err == database.ErrConversationNotFound {
			ctx.Logger.WithError(err).Error("Conversation not found")
			sendJSONError(w, "Conversation not found", http.StatusNotFound)
		} else {
			ctx.Logger.WithError(err).Error("Failed to get conversation details")
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	response := ConversationDetailsResponse{
		ID:           conversation.ID,
		Title:        conversation.Title,
		IsGroup:      conversation.IsGroup,
		UpdatedAt:    conversation.UpdatedAt,
		Participants: convertParticipants(conversation.Participants),
		Messages:     convertMessages(conversation.Messages),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (rt *_router) handleStartConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	ctx.Logger.WithField("userID", userID).Info("Handling start conversation request")

	var req struct {
		Title        string   `json:"title"`
		IsGroup      bool     `json:"isGroup"`
		Participants []string `json:"participants"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Error("Failed to decode request body")
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !req.IsGroup && len(req.Participants) != 1 {
		ctx.Logger.Error("Invalid number of participants for non-group conversation")
		sendJSONError(w, "For non-group conversations, exactly one recipient is required", http.StatusBadRequest)
		return
	}

	if req.IsGroup && len(req.Participants) < 1 {
		ctx.Logger.Error("No participants provided for group conversation")
		sendJSONError(w, "For group conversations, at least one participant is required", http.StatusBadRequest)
		return
	}

	// Add the creator to the participants list if not already present
	participants := append(req.Participants, userID)
	uniqueParticipants := make([]string, 0, len(participants))
	seen := make(map[string]bool)
	for _, p := range participants {
		if !seen[p] {
			seen[p] = true
			uniqueParticipants = append(uniqueParticipants, p)
		}
	}

	conversationID, err := rt.db.StartConversation(userID, req.Title, req.IsGroup, uniqueParticipants)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to start conversation")
		if strings.Contains(err.Error(), "participant with ID") {
			sendJSONError(w, fmt.Sprintf("Invalid participant: %v", err), http.StatusBadRequest)
		} else {
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Fetch the conversation details
	conversation, err := rt.db.GetConversationDetails(conversationID, userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to fetch conversation details")
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := ConversationDetailsResponse{
		ID:           conversation.ID,
		Title:        conversation.Title,
		IsGroup:      conversation.IsGroup,
		UpdatedAt:    conversation.UpdatedAt,
		Participants: convertParticipants(conversation.Participants),
		Messages:     convertMessages(conversation.Messages),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

type sendMessageRequest struct {
	MessageType string `json:"messageType"`
	Content     string `json:"content"`
}

type sendMessageResponse struct {
	MessageID      string    `json:"messageId"`
	ConversationID string    `json:"conversationId"`
	Sender         string    `json:"sender"`
	Content        string    `json:"content"`
	MessageType    string    `json:"messageType"`
	Timestamp      time.Time `json:"timestamp"`
}

func (rt *_router) handleSendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	conversationID := ps.ByName("conversationId")

	// Check if the user is a participant in the conversation
	isParticipant, err := rt.db.IsUserInConversation(userID, conversationID)
	if err != nil {
		if err == database.ErrConversationNotFound {
			sendJSONError(w, "Conversation not found", http.StatusNotFound)
		} else {
			ctx.Logger.WithError(err).Error("Failed to check user participation in conversation")
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}
	if !isParticipant {
		sendJSONError(w, "User is not a participant in this conversation", http.StatusForbidden)
		return
	}

	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Error("Failed to decode request body")
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		sendJSONError(w, "Content is required", http.StatusBadRequest)
		return
	}

	if req.MessageType != "text" && req.MessageType != "photo" {
		sendJSONError(w, "Invalid message type", http.StatusBadRequest)
		return
	}

	messageID, err := rt.db.AddMessage(conversationID, userID, req.MessageType, req.Content)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to add message")
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	senderName, err := rt.db.GetUserNameByID(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to get sender's name")
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := sendMessageResponse{
		MessageID:      messageID,
		ConversationID: conversationID,
		Sender:         senderName,
		Content:        req.Content,
		MessageType:    req.MessageType,
		Timestamp:      time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

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
	if !contains(validStatuses, req.Status) {
		ctx.Logger.WithField("status", req.Status).Error("Invalid status")
		sendJSONError(w, "Invalid status", http.StatusBadRequest)
		return
	}

	err := rt.db.UpdateMessageStatus(messageID, userID, req.Status)
	if err != nil {
		switch err {
		case database.ErrMessageNotFound:
			ctx.Logger.WithError(err).Error("Message not found")
			sendJSONError(w, "Message not found", http.StatusNotFound)
		case database.ErrUnauthorized:
			ctx.Logger.WithError(err).Error("User not authorized to update message status")
			sendJSONError(w, "Unauthorized", http.StatusForbidden)
		default:
			ctx.Logger.WithError(err).Error("Failed to update message status")
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Fetch the updated message
	message, err := rt.db.GetMessageByID(messageID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to fetch updated message")
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert the single message to the response format using the existing conversion function
	convertedMessages := convertMessages([]database.Message{*message})
	if len(convertedMessages) == 0 {
		ctx.Logger.Error("Failed to convert message")
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	response := convertedMessages[0]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type forwardMessageRequest struct {
	OriginalMessageID    string `json:"originalMessageId"`
	TargetConversationID string `json:"targetConversationId"`
}

type forwardMessageResponse struct {
	NewMessageID         string    `json:"newMessageId"`
	OriginalMessageID    string    `json:"originalMessageId"`
	TargetConversationID string    `json:"targetConversationId"`
	Sender               string    `json:"sender"`
	Content              string    `json:"content"`
	MessageType          string    `json:"messageType"`
	Timestamp            time.Time `json:"timestamp"`
}

func (rt *_router) handleForwardMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	ctx.Logger.WithFields(logrus.Fields{
		"userID":    userID,
		"messageID": ps.ByName("messageId"),
	}).Info("Handling forward message request")

	var req forwardMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Error("Failed to decode request body")
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.OriginalMessageID == "" || req.TargetConversationID == "" {
		ctx.Logger.Error("Missing required fields in request")
		sendJSONError(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Forward the message
	newMessage, err := rt.db.ForwardMessage(req.OriginalMessageID, req.TargetConversationID, userID)
	if err != nil {
		var statusCode int
		var errorMessage string
		switch err {
		case database.ErrMessageNotFound:
			statusCode = http.StatusNotFound
			errorMessage = "Original message not found"
		case database.ErrConversationNotFound:
			statusCode = http.StatusNotFound
			errorMessage = "Target conversation not found"
		case database.ErrUnauthorized:
			statusCode = http.StatusForbidden
			errorMessage = "Unauthorized to forward this message"
		default:
			statusCode = http.StatusInternalServerError
			errorMessage = "Internal server error"
		}
		ctx.Logger.WithError(err).Error(errorMessage)
		sendJSONError(w, errorMessage, statusCode)
		return
	}

	response := forwardMessageResponse{
		NewMessageID:         newMessage.ID,
		OriginalMessageID:    req.OriginalMessageID,
		TargetConversationID: req.TargetConversationID,
		Sender:               newMessage.Sender,
		Content:              newMessage.Content,
		MessageType:          newMessage.Type,
		Timestamp:            newMessage.Timestamp,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (rt *_router) handleDeleteMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	ctx.Logger.WithFields(logrus.Fields{
		"userID":    userID,
		"messageID": ps.ByName("messageId"),
	}).Info("Handling delete message request")

	messageID := ps.ByName("messageId")

	// Delete the message
	deletedMessage, err := rt.db.DeleteMessage(messageID, userID)
	if err != nil {
		var statusCode int
		var errorMessage string
		switch err {
		case database.ErrMessageNotFound:
			statusCode = http.StatusNotFound
			errorMessage = "Message not found"
		case database.ErrUnauthorized:
			statusCode = http.StatusForbidden // Changed from StatusUnauthorized to StatusForbidden
			errorMessage = "Forbidden to delete this message"
		default:
			statusCode = http.StatusInternalServerError
			errorMessage = "Internal server error"
			ctx.Logger.WithError(err).Error("Failed to delete message")
		}
		sendJSONError(w, errorMessage, statusCode)
		return
	}

	response := deleteMessageResponse{
		MessageID: deletedMessage.ID,
		Username:  deletedMessage.Sender,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

type deleteMessageResponse struct {
	MessageID string `json:"messageId"`
	Username  string `json:"username"`
}

func (rt *_router) handleAddComment(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	messageID := ps.ByName("messageId")

	ctx.Logger.WithFields(logrus.Fields{
		"messageID": messageID,
		"userID":    userID,
	}).Info("Attempting to add comment or emoji reaction")

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

	runeCount := utf8.RuneCountInString(req.Content)
	if runeCount > 2 || (runeCount == 2 && req.Content != "❤️") {
		ctx.Logger.Error("Invalid emoji: not a valid single character or heart emoji")
		sendJSONError(w, "Invalid emoji: must be a single character or heart emoji", http.StatusBadRequest)
		return
	}

	comment, err := rt.db.AddComment(messageID, userID, req.Content)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to add comment or emoji reaction")
		switch {
		case err.Error() == "user not authorized to comment on this message":
			sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		case err.Error() == "message not found":
			sendJSONError(w, "Message not found", http.StatusNotFound)
		default:
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	ctx.Logger.WithFields(logrus.Fields{
		"commentID": comment.ID,
		"messageID": comment.MessageID,
		"userID":    comment.UserID,
		"content":   comment.Content,
	}).Info("Comment or emoji reaction added successfully")

	response := struct {
		CommentID string    `json:"commentId"`
		MessageID string    `json:"messageId"`
		UserID    string    `json:"userId"`
		Content   string    `json:"content"`
		Timestamp time.Time `json:"timestamp"`
	}{
		CommentID: comment.ID,
		MessageID: comment.MessageID,
		UserID:    comment.UserID,
		Content:   comment.Content,
		Timestamp: comment.Timestamp,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (rt *_router) handleDeleteComment(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	messageID := ps.ByName("messageId")
	commentID := ps.ByName("commentId")

	ctx.Logger.WithFields(logrus.Fields{
		"messageID": messageID,
		"commentID": commentID,
		"userID":    userID,
	}).Info("Attempting to delete comment")

	err := rt.db.DeleteComment(messageID, commentID, userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to delete comment")
		w.Header().Set("Content-Type", "application/json")
		switch {
		case err.Error() == "comment not found":
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Comment not found"})
		case err.Error() == "user not authorized to access this message" || err.Error() == "user not authorized to delete this comment":
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Internal server error"})
		}
		return
	}

	// Get the username of the user who deleted the comment
	username, err := rt.db.GetUserNameByID(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to get username")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Internal server error"})
		return
	}

	ctx.Logger.WithFields(logrus.Fields{
		"messageID": messageID,
		"commentID": commentID,
		"userID":    userID,
		"username":  username,
	}).Info("Comment deleted successfully")

	response := struct {
		MessageID string `json:"messageId"`
		CommentID string `json:"commentId"`
		Username  string `json:"username"`
	}{
		MessageID: messageID,
		CommentID: commentID,
		Username:  username,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
