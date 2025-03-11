# WASAText

WASAText is a project developed for a Web and Software Architecture course. It provides a platform for users to engage in and manage conversations (either alone or in groups) and share photos.

## Backend stack
- Go (Golang) for server-side logic
- SQLite database
- RESTful API design

## API Endpoints

- `POST /session`: User login
- `PUT /user` : Updating username
- `GET /users` : Search for users or get all
- `PUT /user/{userId}`: Update user profile pic
- `GET /conversations`: Get user's conversations
- `POST /conversations`: Start a new conversation
- `GET /conversations/{conversationId}`: Get conversation details
- `POST /conversations/{conversationId}/messages`: Send a message
- `GET /conversations/{conversationId}/messages`: Get messages in a conversation
- `POST /messages/{messageId}/forward`: Forward a message
- `POST /messages/{messageId}/comments`: Add a comment to a message
- `DELETE /messages/{messageId}/comments/{commentId}`: Delete a comment
- `GET /groups`: Get groups of a user
- `POST /groups/{groupId}`: Add a user to group/create one
- `DELETE /groups/{groupId}`: Leave a group (it is deleted if it has no more members)
- `PUT /groups/{groupId}`: Set the group name
- `PATCH /groups/{groupId}`: Set the group photo
