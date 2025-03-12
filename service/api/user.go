package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"errors"

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
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
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
		// At this point headers are already sent, so we can't change the status code
		// Just log the error
	}
 } 

// updatePhotoRequest represents the request body for updating user's photo
type updatePhotoRequest struct {
	PhotoURL string `json:"photoUrl"`
}

// updatePhotoResponse represents the response body for updating user's photo
type updatePhotoResponse struct {
	UserID      string `json:"userId"`
	OldPhotoURL string `json:"oldPhotoUrl"`
	NewPhotoURL string `json:"newPhotoUrl"`
}

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

	var req updatePhotoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Error("Failed to decode request body")
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the photo URL
	if req.PhotoURL == "" {
		ctx.Logger.Warn("Empty photo URL provided")
		sendJSONError(w, "Photo URL cannot be empty", http.StatusBadRequest)
		return
	}

	if _, err := url.ParseRequestURI(req.PhotoURL); err != nil {
		ctx.Logger.WithError(err).Warn("Invalid photo URL provided")
		sendJSONError(w, "Invalid photo URL", http.StatusBadRequest)
		return
	}

	oldPhotoURL, err := rt.db.UpdateUserPhoto(userID, req.PhotoURL)
	if err != nil {
		ctx.Logger.WithFields(logrus.Fields{
			"error":    err,
			"userID":   userID,
			"photoURL": req.PhotoURL,
		}).Error("Failed to update user photo")
		if err == database.ErrUserNotFound {
			sendJSONError(w, "User not found", http.StatusNotFound)
		} else {
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	resp := updatePhotoResponse{
		UserID:      userID,
		OldPhotoURL: oldPhotoURL,
		NewPhotoURL: req.PhotoURL,
	}

	ctx.Logger.WithFields(logrus.Fields{
		"userID":      userID,
		"oldPhotoURL": oldPhotoURL,
		"newPhotoURL": req.PhotoURL,
	}).Info("User photo updated successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
