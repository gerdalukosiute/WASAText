package api

import (
    "net/http"
    "strings"
	"fmt"

    "github.com/julienschmidt/httprouter"
    "github.com/gerdalukosiute/WASAText/service/api/reqcontext"
)

// handleGetMedia handles requests to retrieve media files
func (rt *_router) handleGetMedia(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	mediaID := ps.ByName("mediaId")
	
	// Validate mediaId format
	if !strings.HasPrefix(mediaID, "media") || len(mediaID) < 10 || len(mediaID) > 30 {
		http.Error(w, "Invalid media ID format", http.StatusBadRequest)
		return
	}
	
	// Get the media file from the database
	fileData, mimeType, err := rt.db.GetMediaFile(mediaID)
	if err != nil {
		ctx.Logger.WithError(err).Error("Failed to get media file")
		
		// Check if the media file was not found
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Media file not found", http.StatusNotFound)
			return
		}
		
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	// Set the content type and write the file data
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fileData)))
	w.WriteHeader(http.StatusOK)
	
	if _, err := w.Write(fileData); err != nil {
		ctx.Logger.WithError(err).Error("Failed to write media file to response")
	}
}
