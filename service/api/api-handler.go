package api

import (
	"net/http"
)

// Handler returns an instance of httprouter.Router that handle APIs registered here
func (rt *_router) Handler() http.Handler {
	// Register routes
	rt.router.POST("/session", rt.wrap(rt.handleLogin))
	rt.router.PUT("/user", rt.withAuth(rt.handleUpdateUsername))
	rt.router.GET("/users", rt.withAuth(rt.handleSearchUsers))
	rt.router.PUT("/user/:userId", rt.withAuth(rt.handleUpdateUserPhoto))
	rt.router.GET("/conversations", rt.withAuth(rt.handleGetConversations))
	rt.router.POST("/conversations", rt.withAuth(rt.handleStartConversation))
	rt.router.GET("/conversations/:conversationId", rt.withAuth(rt.handleGetConversationDetails))
	rt.router.POST("/conversations/:conversationId/messages", rt.withAuth(rt.handleSendMessage))
	rt.router.GET("/conversations/:conversationId/messages", rt.withAuth(rt.handleGetMessages))
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
