package main

import (
	"encoding/json"
	"net/http"

	"github.com/mahaj/networking-minor/pkg/auth"
	"github.com/mahaj/networking-minor/pkg/db"
)

type ReadRequest struct {
	OtherUserID string `json:"other_user_id"`
}

func ReadHandler(session *db.Session) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.UserKey).(*auth.Claims)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req ReadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Delete the row from conversation_counters to reset count to 0
		// In ScyllaDB counters, deletion is the way to reset.
		query := `DELETE FROM conversation_counters WHERE user_id = ? AND other_user_id = ?`
		if err := session.Query(query, claims.UserID, req.OtherUserID).Exec(); err != nil {
			http.Error(w, "Failed to reset unread count", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
