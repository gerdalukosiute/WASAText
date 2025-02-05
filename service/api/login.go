package api

import (
	"encoding/json"
	"net/http"

	"github.com/gerdalukosiute/WASAText/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

type loginRequest struct {
	Name string `json:"name"`
}

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

	if len(req.Name) < 3 || len(req.Name) > 16 {
		ctx.Logger.WithField("name", req.Name).Warn("Invalid username length")
		sendJSONError(w, "Invalid username length", http.StatusBadRequest)
		return
	}

	userID, err := rt.db.GetOrCreateUser(req.Name)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to get or create user")
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := loginResponse{Identifier: userID}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
