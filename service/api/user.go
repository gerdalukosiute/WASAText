package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/gerdalukosiute/WASAText/service/api/reqcontext"
	"github.com/gerdalukosiute/WASAText/service/database"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

// updateUsernameRequest represents the request body for updating username
type updateUsernameRequest struct {
	NewName string `json:"newName"`
}

// updateUsernameResponse represents the response body for updating username
type updateUsernameResponse struct {
	Message string `json:"message"`
}

// Username pattern as defined in the API spec: 3-16 characters, alphanumeric with underscores
var usernameRegex = regexp.MustCompile("^[a-zA-Z0-9_]{3,16}$")

// Response message pattern as defined in the API spec
var messageRegex = regexp.MustCompile("^[a-zA-Z0-9_ ]{10,100}$")

// handleUpdateUsername is the HTTP handler for updating a user's username
func (rt *_router) handleUpdateUsername(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	ctx.Logger.Info("Handling update username request")
	var req updateUsernameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Error("Failed to decode request body")
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	// Check if the required field is present and not empty
	if req.NewName == "" {
		ctx.Logger.Warn("Missing required field 'newName'")
		sendJSONError(w, "Missing required field 'newName'", http.StatusBadRequest)
		return
	}
	ctx.Logger.WithField("newName", req.NewName).Info("Received new username")
	// Validate username using the regex pattern from API spec
	// This handles both format and length in one check
	if !usernameRegex.MatchString(req.NewName) {
		ctx.Logger.WithField("newName", req.NewName).Warn("Invalid username format or length")
		sendJSONError(w, "Username must be 3-16 characters and contain only letters, numbers, and underscores", http.StatusBadRequest)
		return
	}

	// Update username in database
	err := rt.db.UpdateUsername(userID, req.NewName)
	if err != nil {
		// Use errors.Is instead of direct comparison for error checking
		if errors.Is(err, database.ErrUserNotFound) {
			ctx.Logger.WithField("userID", userID).Warn("User not found")
			sendJSONError(w, "User not found", http.StatusUnauthorized)
			return
		}
		if errors.Is(err, database.ErrDuplicateUsername) {
			ctx.Logger.WithField("newName", req.NewName).Warn("Username already taken")
			sendJSONError(w, "Username already taken", http.StatusBadRequest)
			return
		}

		// Default case for other errors
		ctx.Logger.WithError(err).Error("Failed to update username")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}
	ctx.Logger.WithFields(logrus.Fields{
		"userID":  userID,
		"newName": req.NewName,
	}).Info("Username updated successfully")
	// Create response message that meets the pattern requirements
	successMessage := "Username successfully updated"

	// Ensure the message meets the pattern requirements
	if !messageRegex.MatchString(successMessage) {
		ctx.Logger.Warn("Response message does not match required pattern, using default")
		successMessage = "Username successfully updated to new value"
	}
	// Send a success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Check error from json.Encoder.Encode
	if err := json.NewEncoder(w).Encode(updateUsernameResponse{Message: successMessage}); err != nil {
		ctx.Logger.WithError(err).Error("Failed to encode response")
	}
}

// handleUpdateUserPhoto handles PUT requests to /user/{userId} for updating profile photos
func (rt *_router) handleUpdateUserPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	ctx.Logger.WithField("userID", userID).Info("Handling update user photo request")

	// Verify that the authenticated user matches the requested user ID
	requestedUserID := ps.ByName("userId")
	if userID != requestedUserID {
		ctx.Logger.WithFields(logrus.Fields{
			"authenticatedUserID": userID,
			"requestedUserID":     requestedUserID,
		}).Warn("Unauthorized attempt to update another user's photo")
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Validate userId format according to API spec
	if !rt.db.IsValidUserID(requestedUserID) {
		ctx.Logger.WithField("userId", requestedUserID).Warn("Invalid user ID format")
		sendJSONError(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	// Limit the file size to 5MB (5242880 bytes)
	r.Body = http.MaxBytesReader(w, r.Body, 5242880)

	// Parse the multipart form
	if err := r.ParseMultipartForm(5242880); err != nil {
		ctx.Logger.WithError(err).Error("Failed to parse multipart form")
		if strings.Contains(err.Error(), "request body too large") {
			sendJSONError(w, "File size exceeds the 5MB limit", http.StatusRequestEntityTooLarge)
		} else {
			sendJSONError(w, "Failed to parse form data", http.StatusBadRequest)
		}
		return
	}

	// Get the file from the form
	file, header, err := r.FormFile("photo")
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to get file from form")
		sendJSONError(w, "No file provided or invalid file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if !rt.db.IsValidImageType(contentType) {
		ctx.Logger.WithField("contentType", contentType).Warn("Invalid file type")
		sendJSONError(w, "Unsupported media type. Only JPEG, PNG, and GIF are allowed", http.StatusUnsupportedMediaType)
		return
	}

	// Read the file data
	fileData, err := io.ReadAll(file)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to read file data")
		sendJSONError(w, "Failed to read file data", http.StatusInternalServerError)
		return
	}

	// Check minimum file size (100 bytes)
	if len(fileData) < 100 {
		ctx.Logger.WithField("fileSize", len(fileData)).Warn("File too small")
		sendJSONError(w, "File too small. Minimum size is 100 bytes", http.StatusBadRequest)
		return
	}

	// Update the user's photo directly in the database
	oldPhotoID, newPhotoID, err := rt.db.UpdateUserPhoto(userID, fileData, contentType)
	if err != nil {
		ctx.Logger.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Failed to update user photo")
		if errors.Is(err, database.ErrUserNotFound) {
			sendJSONError(w, "User not found", http.StatusNotFound)
		} else {
			sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		}
		return
	}

	// Prepare the response according to API spec
	type updatePhotoResponse struct {
		UserID     string `json:"userId"`
		OldPhotoID string `json:"oldPhotoId,omitempty"`
		NewPhotoID string `json:"newPhotoId"`
	}

	resp := updatePhotoResponse{
		UserID:     userID,
		NewPhotoID: newPhotoID,
	}

	// Only include oldPhotoId if there was an old photo
	if oldPhotoID != "" {
		resp.OldPhotoID = oldPhotoID
	}

	ctx.Logger.WithFields(logrus.Fields{
		"userID":     userID,
		"oldPhotoID": oldPhotoID,
		"newPhotoID": newPhotoID,
	}).Info("User photo updated successfully")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		ctx.Logger.WithError(err).Error("Failed to encode JSON response")
		return
	}
}
