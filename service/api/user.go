package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"

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

var usernameRegex = regexp.MustCompile("^[a-zA-Z0-9_]+$")

// handleUpdateUsername is the HTTP handler for updating a user's username
func (rt *_router) handleUpdateUsername(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	ctx.Logger.Info("Handling update username request")

	var req updateUsernameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Error("Failed to decode request body")
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx.Logger.WithField("newName", req.NewName).Info("Received new username")

	// Validate username length
	if len(req.NewName) < 3 || len(req.NewName) > 16 {
		ctx.Logger.WithField("newName", req.NewName).Warn("Invalid username length")
		sendJSONError(w, "Username must be between 3 and 16 characters", http.StatusBadRequest)
		return
	}

	// Validate username format
	if !usernameRegex.MatchString(req.NewName) {
		ctx.Logger.WithField("newName", req.NewName).Warn("Invalid username format")
		sendJSONError(w, "Username must contain only letters, numbers, and underscores", http.StatusBadRequest)
		return
	}

	// Update username in database
	err := rt.db.UpdateUsername(userID, req.NewName)
	if err != nil {
		switch err {
		case database.ErrUserNotFound:
			ctx.Logger.WithField("userID", userID).Warn("User not found")
			sendJSONError(w, "User not found", http.StatusUnauthorized)
		case database.ErrDuplicateUsername:
			ctx.Logger.WithField("newName", req.NewName).Warn("Username already taken")
			sendJSONError(w, "Username already taken", http.StatusBadRequest)
		default:
			ctx.Logger.WithError(err).Error("Failed to update username")
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	ctx.Logger.WithFields(logrus.Fields{
		"userID":  userID,
		"newName": req.NewName,
	}).Info("Username updated successfully")

	// Send a success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updateUsernameResponse{Message: "Username updated successfully"})
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
