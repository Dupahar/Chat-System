package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mahaj/networking-minor/pkg/auth"
	"github.com/mahaj/networking-minor/pkg/db"
)

type Conversation struct {
	UserID      string    `json:"user_id"`
	OtherUserID string    `json:"other_user_id"`
	LastUpdated time.Time `json:"last_updated"`
	UnreadCount int64     `json:"unread_count"`
}

func ConversationsHandler(session *db.Session) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.UserKey).(*auth.Claims)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Query user_conversations
		query := `SELECT user_id, other_user_id, last_updated FROM user_conversations WHERE user_id = ?`
		iter := session.Query(query, claims.UserID).Iter()

		var conversations []Conversation
		var c Conversation
		for iter.Scan(&c.UserID, &c.OtherUserID, &c.LastUpdated) {
			// Fetch unread count for this conversation
			var count int64
			if err := session.Query(`SELECT unread_count FROM conversation_counters WHERE user_id = ? AND other_user_id = ?`, c.UserID, c.OtherUserID).Scan(&count); err == nil {
				c.UnreadCount = count
			}
			conversations = append(conversations, c)
		}

		if err := iter.Close(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(conversations)
	}
}
