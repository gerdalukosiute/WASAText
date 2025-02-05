package database

import (
	"fmt"
)

// SearchUsers searches for users based on a query string
// ideally should also consider pagination
func (db *appdbimpl) SearchUsers(query string) ([]User, error) {
	rows, err := db.c.Query("SELECT id, name FROM users WHERE name LIKE ? LIMIT 20", "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("error searching users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name); err != nil {
			return nil, fmt.Errorf("error scanning user row: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, nil
}
