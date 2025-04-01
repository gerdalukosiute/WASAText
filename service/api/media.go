package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gerdalukosiute/WASAText/service/api/reqcontext"
	"github.com/gerdalukosiute/WASAText/service/database"
	"github.com/julienschmidt/httprouter"
)

// handleGetMedia handles requests to retrieve media files
func (rt *_router) handleGetMedia(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	mediaID := ps.ByName("mediaId")

	// Validate mediaId length only, allowing both media and photo prefixes
	if len(mediaID) < 10 || len(mediaID) > 50 {
		ctx.Logger.WithField("mediaID", mediaID).Warn("Invalid media ID length")
		sendJSONError(w, "Invalid media ID format", http.StatusBadRequest)
		return
	}

	// Get the media file from the database
	fileData, mimeType, err := rt.db.GetMediaFile(mediaID)
	if err != nil {
		ctx.Logger.WithError(err).WithField("mediaID", mediaID).Error("Failed to get media file")

		// Check if the media file was not found
		if errors.Is(err, database.ErrMediaNotFound) || strings.Contains(err.Error(), "not found") {
			sendJSONError(w, "Media file not found", http.StatusNotFound)
			return
		}

		sendJSONError(w, ErrInternalServerMsg, http.StatusInternalServerError)
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
