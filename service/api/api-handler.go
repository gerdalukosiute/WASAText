package api

import (
	"net/http"
)

// Handler returns an instance of httprouter.Router that handle APIs registered here
func (rt *_router) Handler() http.Handler {
	// Register routes
	rt.router.POST("/session", rt.wrap(rt.handleLogin)) // updated
	rt.router.PUT("/user", rt.withAuth(rt.handleUpdateUsername)) // updated
	rt.router.GET("/users", rt.withAuth(rt.handleSearchUsers)) // updated
	rt.router.PUT("/user/:userId", rt.withAuth(rt.handleUpdateUserPhoto)) // updated
	rt.router.GET("/conversations", rt.withAuth(rt.handleGetConversations))  
	// works with current, needs to be retested after some convos present
	rt.router.POST("/conversations", rt.withAuth(rt.handleStartConversation)) 
	// updated, retest aswell

	rt.router.GET("/conversations/:conversationId", rt.withAuth(rt.handleGetConversationDetails))
	// After dealing with messages
	rt.router.POST("/conversations/:conversationId/messages", rt.withAuth(rt.handleSendMessage))
	rt.router.GET("/media/:mediaId", rt.withAuth(rt.handleGetMedia)) // New
	// re-do to handle both pics and texts
	rt.router.PUT("/messages/:messageId/status", rt.withAuth(rt.handleUpdateMessageStatus))
	rt.router.POST("/messages/:messageId/forward", rt.withAuth(rt.handleForwardMessage))
	rt.router.DELETE("/messages/:messageId", rt.withAuth(rt.handleDeleteMessage))
	rt.router.POST("/messages/:messageId/comments", rt.withAuth(rt.handleAddComment))
	rt.router.DELETE("/messages/:messageId/comments/:commentId", rt.withAuth(rt.handleDeleteComment))
	rt.router.GET("/groups", rt.withAuth(rt.handleGetMyGroups))
	rt.router.POST("/groups/:groupId", rt.withAuth(rt.handleAddToGroup))
	rt.router.DELETE("/groups/:groupId", rt.withAuth(rt.handleLeaveGroup))
	rt.router.PUT("/groups/:groupId", rt.withAuth(rt.handleSetGroupName))
	rt.router.PATCH("/groups/:groupId", rt.withAuth(rt.handleSetGroupPhoto))

	// Special routes
	rt.router.GET("/liveness", rt.liveness)

	return rt.router
}
