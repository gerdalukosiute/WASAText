package api

import (
	"encoding/json"
	"net/http"

	"github.com/gerdalukosiute/WASAText/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) handleGetMyGroups(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	ctx.Logger.WithField("userID", userID).Info("Handling get my groups request")

	groups, err := rt.db.GetGroupsForUser(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to fetch groups for user")
		w.Header().Set("Content-Type", "application/json")
		if err.Error() == "user not found" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Internal server error"})
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(groups)
}
