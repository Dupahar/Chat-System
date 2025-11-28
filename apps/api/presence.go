package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
)

type PresenceHandler struct {
	redis *redis.Client
}

func NewPresenceHandler(redisAddr string) *PresenceHandler {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	return &PresenceHandler{redis: rdb}
}

func (h *PresenceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract channel ID from URL path: /channels/{id}/users
	// Simple parsing for now
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 || pathParts[3] != "users" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	channelID := pathParts[2]

	// Query Redis Set
	users, err := h.redis.SMembers(context.Background(), "channel:"+channelID+":users").Result()
	if err != nil {
		log.Printf("Failed to fetch presence for channel %s: %v", channelID, err)
		http.Error(w, "Failed to fetch presence", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
