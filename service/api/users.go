package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gerdalukosiute/WASAText/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

// handleSearchUsers handles GET requests to /users
func (rt *_router) handleSearchUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	ctx.Logger.Info("Handling search users request")

	// Get the search query from the URL parameters
	query := r.URL.Query().Get("q")

	// Trim whitespace from the query
	trimmedQuery := strings.TrimSpace(query)

	// Validate query format if not empty after trimming
	if trimmedQuery != "" {
		// Check if query matches pattern: alphanumeric, underscore, hyphen, max 16 chars
		validQuery := true
		if len(trimmedQuery) > 16 {
			validQuery = false
		} else {
			for _, char := range trimmedQuery {
				if !((char >= 'a' && char <= 'z') ||
					(char >= 'A' && char <= 'Z') ||
					(char >= '0' && char <= '9') ||
					char == '_' || char == '-') {
					validQuery = false
					break
				}
			}
		}

		if !validQuery {
			sendJSONError(w, "Invalid query format. Query must be alphanumeric with underscore or hyphen, max 16 characters", http.StatusBadRequest)
			return
		}
	}

	ctx.Logger.WithFields(logrus.Fields{
		"authenticatedUserID": userID,
		"query":               trimmedQuery, // Log the trimmed query
	}).Info("Authenticated user searching for users")

	// Perform the search using the database with the trimmed query
	users, total, err := rt.db.SearchUsers(trimmedQuery)
	if err != nil {
		ctx.Logger.WithFields(logrus.Fields{
			"authenticatedUserID": userID,
			"error":               err,
		}).Error("Failed to search users")
		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
		return
	}

	ctx.Logger.WithField("usersCount", len(users)).Info("Returning search results")

	// Prepare the response according to API spec
	type UserInfo struct {
		Username       string `json:"username"`
		UserID         string `json:"userId"`
		ProfilePhotoID string `json:"profilePhotoId,omitempty"`
	}

	userInfos := make([]UserInfo, len(users))
	for i, user := range users {
		// Map the database fields to the API response fields
		userInfos[i] = UserInfo{
			Username: user.Name,
			UserID:   user.ID,
			// Only include profilePhotoId if it exists
			ProfilePhotoID: user.PhotoID,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"users": userInfos,
		"total": total,
	}); err != nil {
		ctx.Logger.WithError(err).Error("Failed to encode JSON response")
		return
	}
}
