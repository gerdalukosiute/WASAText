package database

import (
	"database/sql"
	"fmt"
	"strings"
)

// SearchUsers searches for users based on a query string
// Returns all users if query is empty
func (db *appdbimpl) SearchUsers(query string) ([]User, int, error) {
	var rows *sql.Rows
	var err error
	var countQuery string
	var searchQuery string

	// If query is empty or just whitespace, return all users
	if strings.TrimSpace(query) == "" {
		countQuery = "SELECT COUNT(*) FROM users"
		searchQuery = "SELECT id, name, photo_id FROM users LIMIT 1000"
	} else {
		countQuery = "SELECT COUNT(*) FROM users WHERE name LIKE ?"
		searchQuery = "SELECT id, name, photo_id FROM users WHERE name LIKE ? LIMIT 1000"
	}

	// Get total count
	var total int
	var countErr error
	if strings.TrimSpace(query) == "" {
		countErr = db.c.QueryRow(countQuery).Scan(&total)
	} else {
		countErr = db.c.QueryRow(countQuery, "%"+query+"%").Scan(&total)
	}
	if countErr != nil {
		return nil, 0, fmt.Errorf("error counting users: %w", countErr)
	}

	// Execute search query
	if strings.TrimSpace(query) == "" {
		rows, err = db.c.Query(searchQuery)
	} else {
		rows, err = db.c.Query(searchQuery, "%"+query+"%")
	}

	if err != nil {
		return nil, 0, fmt.Errorf("error searching users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var photoID sql.NullString
		if err := rows.Scan(&user.ID, &user.Name, &photoID); err != nil {
			return nil, 0, fmt.Errorf("error scanning user row: %w", err)
		}
		if photoID.Valid {
			user.PhotoID = photoID.String
		}
		users = append(users, user)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, total, nil
}
