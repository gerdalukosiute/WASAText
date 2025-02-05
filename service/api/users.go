package api

import (
	"encoding/json"
	"net/http"

	"github.com/gerdalukosiute/WASAText/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

// handleSearchUsers handles GET requests to /users
func (rt *_router) handleSearchUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	ctx.Logger.Info("Handling search users request")

	// Get the search query from the URL parameters
	query := r.URL.Query().Get("q")

	ctx.Logger.WithFields(logrus.Fields{
		"authenticatedUserID": userID,
		"query":               query,
	}).Info("Authenticated user searching for users")

	// Perform the search using the database
	users, err := rt.db.SearchUsers(query)
	if err != nil {
		ctx.Logger.WithFields(logrus.Fields{
			"authenticatedUserID": userID,
			"error":               err,
		}).Error("Failed to search users")
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	ctx.Logger.WithField("usersCount", len(users)).Info("Returning search results")

	// Prepare the response
	type UserInfo struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	userInfos := make([]UserInfo, len(users))
	for i, user := range users {
		userInfos[i] = UserInfo{ID: user.ID, Name: user.Name}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": userInfos,
		"query": query,
	})
}
