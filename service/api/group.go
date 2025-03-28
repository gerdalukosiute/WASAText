package api

import (
	"encoding/json"
	"net/http"
	"errors"
	"fmt"
	"time"

	"github.com/gerdalukosiute/WASAText/service/api/reqcontext"
	"github.com/gerdalukosiute/WASAText/service/database"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

// Updated
func (rt *_router) handleAddToGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
	groupID := ps.ByName("groupId")


	ctx.Logger.WithFields(logrus.Fields{
		"groupID": groupID,
		"userID":  userID,
	}).Info("Handling add to group request")


	// Validate request body
	var req struct {
		Usernames []string `json:"usernames"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Warn("Invalid request body")
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}


	// Validate required fields
	if len(req.Usernames) == 0 {
		ctx.Logger.Warn("No usernames provided")
		sendJSONError(w, "Usernames are required", http.StatusBadRequest)
		return
	}


	// Add users to group
	result, err := rt.db.AddUsersToGroup(groupID, userID, req.Usernames)
	if err != nil {
		// Add detailed error logging
		ctx.Logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"errorType":  fmt.Sprintf("%T", err),
			"groupID":    groupID,
			"userID":     userID,
			"usernames":  req.Usernames,
			"isGroupNotFound": errors.Is(err, database.ErrGroupNotFound),
			"isUnauthorized":  errors.Is(err, database.ErrUnauthorized),
		}).Error("Failed to add users to group")
		
		// Direct string comparison as a fallback
		if errors.Is(err, database.ErrGroupNotFound) || err.Error() == "group not found" {
			ctx.Logger.Warn("Attempt to add users to non-existent group")
			sendJSONError(w, "Group not found", http.StatusNotFound)
			return
		} else if errors.Is(err, database.ErrUnauthorized) || err.Error() == "user unauthorized" {
			ctx.Logger.Warn("Unauthorized attempt to add users to group")
			sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
			return
		} else {
			ctx.Logger.WithError(err).Error("Internal server error when adding users to group")
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}


	// Create response according to API documentation
	response := struct {
		GroupName         string `json:"groupName"`
		GroupID           string `json:"groupId"`
		AddedUsers        []struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		} `json:"addedUsers"`
		FailedUsers       []string `json:"failedUsers"`
		UpdatedMemberCount int      `json:"updatedMemberCount"`
		AddedBy           struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		} `json:"addedBy"`
		Timestamp         string   `json:"timestamp"`
	}{
		GroupName:         result.GroupName,
		GroupID:           result.GroupID,
		AddedUsers:        make([]struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		}, len(result.AddedUsers)),
		FailedUsers:       result.FailedUsers,
		UpdatedMemberCount: result.UpdatedMemberCount,
		AddedBy: struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		}{
			Username: result.AddedBy.Name,
			UserID:   result.AddedBy.ID,
		},
		Timestamp:         result.Timestamp.Format(time.RFC3339),
	}


	// Copy added users to response
	for i, user := range result.AddedUsers {
		response.AddedUsers[i] = struct {
			Username string `json:"username"`
			UserID   string `json:"userId"`
		}{
			Username: user.Username,
			UserID:   user.UserID,
		}
	}


	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		ctx.Logger.WithError(err).Error("Failed to encode response")
	}
}

// Updated
func (rt *_router) handleLeaveGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
   groupID := ps.ByName("groupId")


   ctx.Logger.WithFields(logrus.Fields{
       "groupID": groupID,
       "userID":  userID,
   }).Info("Handling leave group request")


   username, isGroupDeleted, remainingMemberCount, err := rt.db.LeaveGroup(groupID, userID)
   if err != nil {
       ctx.Logger.WithError(err).Error("Failed to leave group")
      
       var statusCode int
       var errorMessage string
      
       if errors.Is(err, database.ErrUnauthorized) {
           statusCode = http.StatusForbidden
           errorMessage = "You are not a member of this group"
       } else if errors.Is(err, database.ErrGroupNotFound) {
           statusCode = http.StatusNotFound
           errorMessage = "Group not found"
       } else {
           statusCode = http.StatusInternalServerError
           errorMessage = "Internal server error"
       }
      
       sendJSONError(w, errorMessage, statusCode)
       return
   }


   // Create the response according to the API documentation
   response := struct {
       GroupID              string    `json:"groupId"`
       User                 struct {
           Username string `json:"username"`
           UserID   string `json:"userId"`
       } `json:"user"`
       IsGroupDeleted       bool      `json:"isGroupDeleted"`
       RemainingMemberCount int       `json:"remainingMemberCount"`
       LeftAt               string    `json:"leftAt"`
   }{
       GroupID:              groupID,
       User: struct {
           Username string `json:"username"`
           UserID   string `json:"userId"`
       }{
           Username: username,
           UserID:   userID,
       },
       IsGroupDeleted:       isGroupDeleted,
       RemainingMemberCount: remainingMemberCount,
       LeftAt:               time.Now().Format(time.RFC3339),
   }


   w.Header().Set("Content-Type", "application/json")
   w.WriteHeader(http.StatusOK)
   if err := json.NewEncoder(w).Encode(response); err != nil {
       ctx.Logger.WithError(err).Error("Failed to encode response")
   }
}

// Updated
func (rt *_router) handleSetGroupName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext, userID string) {
   groupID := ps.ByName("groupId")


   ctx.Logger.WithFields(logrus.Fields{
       "groupID": groupID,
       "userID":  userID,
   }).Info("Handling set group name request")


   // Validate request body
   var req struct {
       GroupName string `json:"groupName"`
   }
   if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
       ctx.Logger.WithError(err).Warn("Invalid request body")
       sendJSONError(w, "Invalid request body", http.StatusBadRequest)
       return
   }


   // Validate required fields
   if req.GroupName == "" {
       ctx.Logger.Warn("No group name provided")
       sendJSONError(w, "Group name is required", http.StatusBadRequest)
       return
   }


   // Update the group name
   oldGroupName, newGroupName, memberCount, err := rt.db.SetGroupName(groupID, userID, req.GroupName)
   if err != nil {
       ctx.Logger.WithError(err).Error("Failed to set group name")
      
       var statusCode int
       var errorMessage string
      
       if errors.Is(err, database.ErrUnauthorized) {
           statusCode = http.StatusForbidden
           errorMessage = "No permission to update"
       } else if errors.Is(err, database.ErrGroupNotFound) {
           statusCode = http.StatusNotFound
           errorMessage = "Group not found"
       } else if errors.Is(err, database.ErrInvalidGroupName) {
           statusCode = http.StatusBadRequest
           errorMessage = "Invalid group name format"
       } else if errors.Is(err, database.ErrNameAlreadyTaken) {
           statusCode = http.StatusConflict
           errorMessage = "Group with this name already exists"
       } else {
           statusCode = http.StatusInternalServerError
           errorMessage = "Internal server error"
       }
      
       sendJSONError(w, errorMessage, statusCode)
       return
   }


   // Get the username for the response
   username, err := rt.db.GetUserNameByID(userID)
   if err != nil {
       ctx.Logger.WithError(err).Error("Failed to get username")
       sendJSONError(w, "Internal server error", http.StatusInternalServerError)
       return
   }


   // Create the response according to the API documentation
   response := struct {
       GroupID      string    `json:"groupId"`
       OldGroupName string    `json:"oldGroupName"`
       NewGroupName string    `json:"newGroupName"`
       UpdatedBy    struct {
           Username string `json:"username"`
           UserID   string `json:"userId"`
       } `json:"updatedBy"`
       UpdatedAt    string    `json:"updatedAt"`
       MemberCount  int       `json:"memberCount"`
   }{
       GroupID:      groupID,
       OldGroupName: oldGroupName,
       NewGroupName: newGroupName,
       UpdatedBy: struct {
           Username string `json:"username"`
           UserID   string `json:"userId"`
       }{
           Username: username,
           UserID:   userID,
       },
       UpdatedAt:    time.Now().Format(time.RFC3339),
       MemberCount:  memberCount,
   }


   w.Header().Set("Content-Type", "application/json")
   w.WriteHeader(http.StatusOK)
   if err := json.NewEncoder(w).Encode(response); err != nil {
       ctx.Logger.WithError(err).Error("Failed to encode response")
   }
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
