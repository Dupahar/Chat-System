package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/mahaj/networking-minor/pkg/db"
)

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Allow all for dev, or specific origin
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	scyllaHostsStr := os.Getenv("SCYLLA_HOSTS")
	if scyllaHostsStr == "" {
		scyllaHostsStr = "localhost:9042"
	}
	scyllaHosts := strings.Split(scyllaHostsStr, ",")
	keyspace := "chat"

	session, err := db.NewSession(scyllaHosts, keyspace)
	if err != nil {
		log.Fatalf("Failed to connect to ScyllaDB: %v", err)
	}
	defer session.Close()

	log.Println("API Service Starting on :8081...")

	// Public endpoint
	http.Handle("/login", CORSMiddleware(http.HandlerFunc(LoginHandler)))

	// Protected endpoint
	historyHandler := NewHistoryHandler(session)
	http.Handle("/history", CORSMiddleware(AuthMiddleware(historyHandler)))

	// Presence endpoint
	// Route: /channels/{id}/users
	// We need a router for path params, but for now we can use a prefix handler and parse manually in handler
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	presenceHandler := NewPresenceHandler(redisAddr)
	http.Handle("/channels/", CORSMiddleware(AuthMiddleware(presenceHandler)))

	// Conversations endpoint
	http.Handle("/conversations", CORSMiddleware(AuthMiddleware(ConversationsHandler(session))))
	http.Handle("/conversations/read", CORSMiddleware(AuthMiddleware(ReadHandler(session))))

	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}
