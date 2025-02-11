package database

import (
	"fmt"
)

func (db *appdbimpl) GetGroupsForUser(userID string) ([]Group, error) {
	rows, err := db.c.Query(`
		SELECT c.id, c.title
		FROM conversations c
		JOIN user_conversations uc ON c.id = uc.conversation_id
		WHERE uc.user_id = ? AND c.is_group = 1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying groups: %w", err)
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			return nil, fmt.Errorf("error scanning group: %w", err)
		}
		groups = append(groups, g)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating groups: %w", err)
	}

	return groups, nil
}
