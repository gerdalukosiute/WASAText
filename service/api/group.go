package api

import (
	"encoding/json"
	"net/http"

	"github.com/gerdalukosiute/WASAText/service/api/reqcontext"
	"github.com/gerdalukosiute/WASAText/service/database"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) handleAddToGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	groupID := ps.ByName("groupId")

	// Validate groupID
	if _, err := uuid.Parse(groupID); err != nil {
		ctx.Logger.WithError(err).Warn("Invalid group ID format")
		sendJSONError(w, "Invalid group ID format", http.StatusBadRequest)
		return
	}

	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Warn("Invalid request body")
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := rt.db.AddUserToGroup(groupID, userID, req.Username)
	if err != nil {
		switch err {
		case database.ErrUnauthorized:
			ctx.Logger.Warn("Unauthorized attempt to add user to group")
			sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		case database.ErrGroupNotFound:
			ctx.Logger.Warn("Attempt to add user to non-existent group")
			sendJSONError(w, "Group not found", http.StatusNotFound)
		case database.ErrUserNotFound:
			ctx.Logger.Warn("Attempt to add non-existent user to group")
			sendJSONError(w, "User not found", http.StatusNotFound)
		case database.ErrUserAlreadyInGroup:
			ctx.Logger.Warn("Attempt to add user already in group")
			sendJSONError(w, "User is already a member of the group", http.StatusConflict)
		default:
			ctx.Logger.WithError(err).Error("Failed to add user to group")
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	response := struct {
		GroupID  string `json:"groupId"`
		Username string `json:"username"`
	}{
		GroupID:  groupID,
		Username: req.Username,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (rt *_router) handleLeaveGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	groupID := ps.ByName("groupId")

	username, isGroupDeleted, err := rt.db.LeaveGroup(groupID, userID)
	if err != nil {
		switch err {
		case database.ErrUnauthorized:
			ctx.Logger.Warn("Unauthorized attempt to leave group")
			sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		case database.ErrGroupNotFound:
			ctx.Logger.Warn("Attempt to leave non-existent group")
			sendJSONError(w, "Group not found", http.StatusNotFound)
		default:
			ctx.Logger.WithError(err).Error("Failed to leave group")
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	response := struct {
		GroupID        string `json:"groupId"`
		Username       string `json:"username"`
		IsGroupDeleted bool   `json:"isGroupDeleted"`
	}{
		GroupID:        groupID,
		Username:       username,
		IsGroupDeleted: isGroupDeleted,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (rt *_router) handleSetGroupName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	groupID := ps.ByName("groupId")

	var req struct {
		GroupName string `json:"groupName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Warn("Invalid request body")
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	oldGroupName, newGroupName, err := rt.db.SetGroupName(groupID, userID, req.GroupName)
	if err != nil {
		switch err {
		case database.ErrUnauthorized:
			ctx.Logger.Warn("Unauthorized attempt to set group name")
			sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		case database.ErrGroupNotFound:
			ctx.Logger.Warn("Attempt to set name of non-existent group")
			sendJSONError(w, "Group not found", http.StatusNotFound)
		default:
			ctx.Logger.WithError(err).Error("Failed to set group name")
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	response := struct {
		GroupID      string `json:"groupId"`
		OldGroupName string `json:"oldGroupName"`
		NewGroupName string `json:"newGroupName"`
	}{
		GroupID:      groupID,
		OldGroupName: oldGroupName,
		NewGroupName: newGroupName,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (rt *_router) handleSetGroupPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	groupID := ps.ByName("groupId")

	var req struct {
		GroupPhoto string `json:"groupPhoto"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Warn("Invalid request body")
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	oldGroupPhoto, newGroupPhoto, err := rt.db.SetGroupPhoto(groupID, userID, req.GroupPhoto)
	if err != nil {
		switch err {
		case database.ErrUnauthorized:
			ctx.Logger.Warn("Unauthorized attempt to set group photo")
			sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		case database.ErrGroupNotFound:
			ctx.Logger.Warn("Attempt to set photo of non-existent group")
			sendJSONError(w, "Group not found", http.StatusNotFound)
		default:
			ctx.Logger.WithError(err).Error("Failed to set group photo")
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	response := struct {
		GroupID       string `json:"groupId"`
		OldGroupPhoto string `json:"oldGroupPhoto"`
		NewGroupPhoto string `json:"newGroupPhoto"`
	}{
		GroupID:       groupID,
		OldGroupPhoto: oldGroupPhoto,
		NewGroupPhoto: newGroupPhoto,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
