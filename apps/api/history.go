package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/mahaj/networking-minor/pkg/auth"
	"github.com/mahaj/networking-minor/pkg/db"
	"github.com/mahaj/networking-minor/pkg/model"
)

type HistoryHandler struct {
	db *db.Session
}

func NewHistoryHandler(session *db.Session) *HistoryHandler {
	return &HistoryHandler{db: session}
}

func (h *HistoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	channelID := r.URL.Query().Get("channel_id")
	if channelID == "" {
		channelID = "general" // Default to general
	}

	var messages []model.Message
	// Query by channel_id (Partition Key)
	iter := h.db.Query("SELECT channel_id, id, user_id, content, timestamp FROM messages WHERE channel_id = ?", channelID).Iter()

	var id int64
	var userID, content, chID string
	var timestamp time.Time

	for iter.Scan(&chID, &id, &userID, &content, &timestamp) {
		messages = append(messages, model.Message{
			ID:        id,
			ChannelID: chID,
			UserID:    userID,
			Content:   content,
			Timestamp: timestamp,
		})
	}

	if err := iter.Close(); err != nil {
		log.Printf("Failed to iterate messages: %v", err)
		http.Error(w, "Failed to retrieve history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

type LoginRequest struct {
	UserID string `json:"user_id"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	token, err := auth.GenerateToken(req.UserID)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{Token: token})
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Remove "Bearer " prefix if present
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add user_id to context (optional, for future use)
		log.Printf("Authenticated user: %s", claims.UserID)

		next.ServeHTTP(w, r)
	})
}
