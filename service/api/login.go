package api

import (
	"encoding/json"
	"regexp"
	"net/http"

	"github.com/gerdalukosiute/WASAText/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

// loginRequest describes the data received in a login request.
type loginRequest struct {
	Name string `json:"name"` 
}

// loginResponse describes the data sent as response to a login request.
type loginResponse struct {
	Identifier string `json:"identifier"` 
}

// handleLogin is the HTTP endpoint that handles user login
func (rt *_router) handleLogin(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Error("Failed to decode request body")
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate name length
	if len(req.Name) < 3 || len(req.Name) > 16 {
		ctx.Logger.WithField("name", req.Name).Warn("Invalid name length")
		sendJSONError(w, "Name must be between 3 and 16 characters", http.StatusBadRequest)
		return
	}

	// Validate name pattern: alphanumeric characters, underscores, and hyphens
	namePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]{3,16}$`)
	if !namePattern.MatchString(req.Name) {
		ctx.Logger.WithField("name", req.Name).Warn("Invalid name format")
		sendJSONError(w, "Name must contain only alphanumeric characters, underscores, and hyphens", http.StatusBadRequest)
		return
	}

	// Get or create user with a 12-character identifier
	userID, err := rt.db.GetOrCreateUser(req.Name)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to get or create user")
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create response
	resp := loginResponse{
		Identifier: userID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		ctx.Logger.WithError(err).Error("Failed to encode response")
		return
	}
}
