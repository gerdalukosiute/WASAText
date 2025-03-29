package api

import (
	"encoding/json"
	"net/http"


	"github.com/gerdalukosiute/WASAText/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

// authenticatedHandler is the signature for handlers that require authentication
type authenticatedHandler func(http.ResponseWriter, *http.Request, httprouter.Params, reqcontext.RequestContext, string)

// withAuth wraps a handler requiring authentication
func (rt *_router) withAuth(handler authenticatedHandler) httprouter.Handle {
	return rt.wrap(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
		// Get user ID from header
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			http.Error(w, "Unauthorized: Missing user identifier", http.StatusUnauthorized)
			return
		}

		handler(w, r, ps, ctx, userID)
	})
}

// function to send JSON-formatted error responses
func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errResp := map[string]string{"error": message}
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		http.Error(w, ErrInternalServerMsg, http.StatusInternalServerError)
	}
}