package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// AppDatabase is the high level interface for the DB
type AppDatabase interface {
	GetOrCreateUser(name string) (string, error)
	UpdateUsername(userID string, newName string) error
	SearchUsers(query string) ([]User, error)
	UpdateUserPhoto(userID string, photoURL string) (string, error)
	GetUserConversations(userID string) ([]Conversation, error)
	StartConversation(initiatorID string, title string, isGroup bool, participants []string) (string, error)
	GetConversationDetails(conversationID, userID string) (*ConversationDetails, error)
	GetMessages(conversationID string, limit int, before time.Time) ([]Message, error)
	AddMessage(conversationID, senderID, messageType, content string) (string, error)
	GetUserNameByID(userID string) (string, error)
	GetComments(messageID string) ([]Comment, error)
	ForwardMessage(originalMessageID, targetConversationID, userID string) (*Message, error)
	DeleteMessage(messageID, userID string) (*Message, error)
	AddComment(messageID, userID, content string) (*Comment, error)
	DeleteComment(messageID, commentID, userID string) error
	GetGroupsForUser(userID string) ([]Group, error)
	AddUserToGroup(groupID, adderID, username string, title string) error
	LeaveGroup(groupID string, userID string) (username string, isGroupDeleted bool, err error)
	SetGroupName(groupID string, userID string, newName string) (oldName string, updatedName string, err error)
	SetGroupPhoto(groupID string, userID string, newPhotoURL string) (oldPhotoURL string, updatedPhotoURL string, err error)
	UserExists(userID string) (bool, error)
	IsUserInConversation(userID, conversationID string) (bool, error)
	Ping() error
}

// User represents a user in the database
type User struct {
	ID       string
	Name     string
	PhotoURL string
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
	ProfilePhoto string
	Participants []Participant
	Messages     []Message
	UpdatedAt    time.Time
}

// Participant represents a user participating in a conversation
type Participant struct {
	ID   string
	Name string
}

// Message represents a message in a conversation
type Message struct {
	ID        string
	SenderID  string
	Sender    string
	Type      string
	Content   string
	Icon      string
	Timestamp time.Time
	Status    string
	Comments  []Comment
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

// Conversation represents a summary of a conversation in the database
type Conversation struct {
	ID           string
	Title        string
	ProfilePhoto string
	IsGroup      bool
	LastMessage  Message
	UpdatedAt    time.Time
}

// Error definitions
var (
	ErrUserNotFound         = errors.New("user not found")
	ErrDuplicateUsername    = errors.New("username already taken")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageNotFound      = errors.New("message not found")
	ErrUnauthorized         = errors.New("user unauthorized")
	ErrGroupNotFound        = errors.New("group not found")
	ErrInvalidGroupName     = errors.New("invalid group name")
	ErrUserAlreadyInGroup   = fmt.Errorf("user is already a member of the group")
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
			photo_url TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			title TEXT,
			profile_photo TEXT,
			is_group BOOLEAN NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			type TEXT NOT NULL,
			content TEXT NOT NULL,
			icon TEXT,
			created_at DATETIME NOT NULL,
			status TEXT NOT NULL,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id),
			FOREIGN KEY (sender_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_conversations (
			user_id TEXT NOT NULL,
			conversation_id TEXT NOT NULL,
			PRIMARY KEY (user_id, conversation_id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (conversation_id) REFERENCES conversations(id)
		)`,
		`CREATE TABLE IF NOT EXISTS reactions (
			message_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			reaction TEXT NOT NULL,
			PRIMARY KEY (message_id, user_id),
			FOREIGN KEY (message_id) REFERENCES messages(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
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
	}

	for _, table := range tables {
		_, err := db.Exec(table)
		if err != nil {
			return fmt.Errorf("error creating table: %w", err)
		}
	}

	// Add 'status' column to messages table if it doesn't exist
	_, err := db.Exec(`ALTER TABLE messages ADD COLUMN status TEXT`)
	if err != nil {
		// If the error is not "duplicate column name", return the error
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("error adding status column: %w", err)
		}
		// If the error is "duplicate column name", the column already exists, so we can ignore it
	}

	logrus.Info("Database tables created or already exist")
	return nil
}

func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}
