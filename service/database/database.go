package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
	"math/rand"

	"github.com/sirupsen/logrus"
)

// AppDatabase is the high level interface for the DB
type AppDatabase interface {
	GetOrCreateUser(name string) (string, error)
	UpdateUsername(userID string, newName string) error
	SearchUsers(query string) ([]User, int, error)
	UpdateUserPhoto(userID string, photoID string) (string, error)
	GetUserConversations(userID string) ([]Conversation, int, error)
	StartConversation(initiatorID string, recipientIDs []string, title string, isGroup bool) (string, error)
	GetUserIDByName(name string) (string, error)
	GetExistingConversation(userID1, userID2 string) (string, bool, error)
	GenerateConversationID() (string, error)
	AddMessage(conversationID, senderID, messageType, content string, contentType string, parentMessageID *string) (string, error)  
	ValidateParentMessage(messageID, conversationID string) (bool, error) 
	IsUserInConversation(userID, conversationID string) (bool, error)
	GetUserNameByID(userID string) (string, error)
	GenerateMessageID() (string, error) 
    StoreMediaFile(fileData []byte, mimeType string) (string, error)
    GetMediaFile(mediaID string) ([]byte, string, error) 
	GetConversationDetails(conversationID, userID string) (*ConversationDetails, error) // not updated
	GetComments(messageID string) ([]Comment, error) // not updated
	ForwardMessage(originalMessageID, targetConversationID, userID string) (*ForwardedMessage, error)
	IsUserAuthorized(userID string, messageID string) (bool, error) 
	ConversationExists(conversationID string) (bool, error)
	DeleteMessage(messageID, userID string) (*Message, error) // not updated
	AddComment(messageID, userID, content string) (*Comment, error) 
	DeleteComment(messageID, commentID, userID string) error 
	GetGroupsForUser(userID string) ([]Group, error) // not updated
	AddUserToGroup(groupID, adderID, username string) error // not updated
	LeaveGroup(groupID string, userID string) (username string, isGroupDeleted bool, err error) // not updated
	SetGroupName(groupID string, userID string, newName string) (oldName string, updatedName string, err error) // not updated
	SetGroupPhoto(groupID string, userID string, newPhotoURL string) (oldPhotoURL string, updatedPhotoURL string, err error) // not updated
	UserExists(userID string) (bool, error)
	UpdateMessageStatus(messageID, userID, newStatus string) (*MessageStatusUpdate, error)
	GetMessageByID(messageID string) (*Message, error)
	Ping() error
}

// User represents a user in the database
type User struct {
	ID       string
	Name     string
	PhotoID  string
}

// Group structure representation
type Group struct {
	ID   string `json:"groupId"`
	Name string `json:"groupName"`
}

// ConversationDetails represents the full details of a conversation
type ConversationDetails struct {
	ID           string
	Title        string
	IsGroup      bool
	UpdatedAt    time.Time
	Participants []Participant
	Messages     []Message
}

// Participant represents a user participating in a conversation
type Participant struct {
	ID   string
	Name string
}

// Message struct represents a message
type Message struct {
	ID               string
	SenderID         string
	Sender           string
	Type             string
	Content          string
	ContentType      string
	Icon             string
	Timestamp        time.Time
	Status           string
	Comments         []Comment
	ParentMessageID  *string
	IsForwarded      bool
	OriginalSender   *User
	OriginalTimestamp time.Time
}


// New struct for forwarded message details
type ForwardedMessage struct {
	ID               string
	SenderID         string
	Sender           string
	Type             string
	Content          string
	ContentType      string
	Timestamp        time.Time
	Status           string
	OriginalSender   User
	OriginalTimestamp time.Time
}

// Comment represents a comment on a message
type Comment struct {
	ID        string
	MessageID string
	UserID    string
	Username  string
	Content   string
	Timestamp time.Time
}

// Conversation represents a summary of a conversation in the database (Updated)
type Conversation struct {
	ID           string
	Title        string
	CreatedAt    time.Time
	ProfilePhoto *string
	IsGroup      bool
	LastMessage  struct {
		Type      string
		Content   string
		Timestamp time.Time
	}
}

// MessageStatusUpdate represents the result of a message status update
type MessageStatusUpdate struct {
	MessageID      string
	Status         string
	UpdatedBy      User
	UpdatedAt      time.Time
	ConversationID string
}

// Error definitions
var (
	// Current used in users, user, conversations
	ErrUserNotFound         = errors.New("user not found") 
	ErrDuplicateUsername    = errors.New("username already taken") 
    ErrUnauthorized         = errors.New("user unauthorized")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageNotFound      = errors.New("message not found")
	ErrGroupNotFound        = errors.New("group not found")
	ErrInvalidGroupName     = errors.New("invalid group name")
	ErrUserAlreadyInGroup   = fmt.Errorf("user is already a member of the group")
	ErrInvalidNameLength = errors.New("invalid name length")
	ErrInvalidNameFormat = errors.New("invalid name format")
	ErrNameAlreadyTaken  = errors.New("name already taken")
)

type appdbimpl struct {
	c *sql.DB
}

// New returns a new instance of AppDatabase based on the SQLite connection `db`.
// `db` is required - an error will be returned if `db` is `nil`.
func New(db *sql.DB) (AppDatabase, error) {
	if db == nil {
		return nil, errors.New("database is required when building a AppDatabase")
	}

	// Seed the random number generator
    rand.Seed(time.Now().UnixNano())


	// Check if tables exist. If not, create them.
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("error creating database structure: %w", err)
	}

	return &appdbimpl{
		c: db,
	}, nil
}

func createTables(db *sql.DB) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			photo_id TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			title TEXT,
			profile_photo TEXT,
			is_group BOOLEAN NOT NULL,
			created_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			type TEXT NOT NULL,
			content TEXT NOT NULL,
			content_type TEXT,
			icon TEXT,
			created_at DATETIME NOT NULL,
			status TEXT NOT NULL,
			parent_message_id TEXT,
			is_forwarded BOOLEAN DEFAULT 0,
			original_sender_id TEXT,
			original_timestamp DATETIME,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id),
			FOREIGN KEY (sender_id) REFERENCES users(id),
			FOREIGN KEY (parent_message_id) REFERENCES messages(id),
			FOREIGN KEY (original_sender_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS message_read_status (
    		message_id TEXT,
    		user_id TEXT,
    		status TEXT,
    		PRIMARY KEY (message_id, user_id),
    		FOREIGN KEY (message_id) REFERENCES messages(id),
    		FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_conversations (
			user_id TEXT NOT NULL,
			conversation_id TEXT NOT NULL,
			PRIMARY KEY (user_id, conversation_id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (conversation_id) REFERENCES conversations(id)
		)`,
		`CREATE TABLE IF NOT EXISTS comments (
			id TEXT PRIMARY KEY,
			message_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (message_id) REFERENCES messages(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS group_members (
			group_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			PRIMARY KEY (group_id, user_id),
			FOREIGN KEY (group_id) REFERENCES groups(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		// Added new media_files table
		`CREATE TABLE IF NOT EXISTS media_files (
		id TEXT PRIMARY KEY,
		file_data BLOB NOT NULL,
		mime_type TEXT NOT NULL,
		created_at DATETIME NOT NULL
		)`,
	}

	for _, table := range tables {
		_, err := db.Exec(table)
		if err != nil {
			return fmt.Errorf("error creating table: %w", err)
		}
	}

	logrus.Info("Database tables created or already exist")
	return nil
}

func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}
