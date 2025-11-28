package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/mahaj/networking-minor/pkg/db"
)

func main() {
	kafkaBrokersStr := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokersStr == "" {
		kafkaBrokersStr = "localhost:19092"
	}
	brokers := strings.Split(kafkaBrokersStr, ",")

	scyllaHostsStr := os.Getenv("SCYLLA_HOSTS")
	if scyllaHostsStr == "" {
		scyllaHostsStr = "localhost:9042"
	}
	scyllaHosts := strings.Split(scyllaHostsStr, ",")

	topic := "chat-messages"
	groupID := "messaging-service-group"
	keyspace := "chat"

	// Initialize ScyllaDB
	// Note: In production, schema creation should be handled by migration tools
	// For this MVP, we'll try to create keyspace/table if not exists (requires a session without keyspace first)

	// Connect to system keyspace to create chat keyspace
	sysSession, err := db.NewSession(scyllaHosts, "system")
	if err != nil {
		log.Fatalf("Failed to connect to ScyllaDB system keyspace: %v", err)
	}

	err = sysSession.Query(`CREATE KEYSPACE IF NOT EXISTS chat WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 }`).Exec()
	if err != nil {
		log.Fatalf("Failed to create keyspace: %v", err)
	}
	sysSession.Close()

	// Connect to chat keyspace
	session, err := db.NewSession(scyllaHosts, keyspace)
	if err != nil {
		log.Fatalf("Failed to connect to ScyllaDB chat keyspace: %v", err)
	}
	defer session.Close()

	// Create messages table
	err = session.Query(`CREATE TABLE IF NOT EXISTS messages (
		channel_id text,
		id bigint,
		user_id text,
		content text,
		timestamp timestamp,
		PRIMARY KEY (channel_id, id)
	) WITH CLUSTERING ORDER BY (id DESC)`).Exec()
	if err != nil {
		log.Fatalf("Failed to create messages table: %v", err)
	}

	// Create user_conversations table
	err = session.Query(`CREATE TABLE IF NOT EXISTS user_conversations (
		user_id text,
		other_user_id text,
		last_updated timestamp,
		PRIMARY KEY (user_id, other_user_id)
	)`).Exec()
	if err != nil {
		log.Fatalf("Failed to create user_conversations table: %v", err)
	}

	// Create conversation_counters table
	err = session.Query(`CREATE TABLE IF NOT EXISTS conversation_counters (
		user_id text,
		other_user_id text,
		unread_count counter,
		PRIMARY KEY (user_id, other_user_id)
	)`).Exec()
	if err != nil {
		log.Fatalf("Failed to create conversation_counters table: %v", err)
	}

	consumer := NewConsumer(brokers, topic, groupID, session)
	defer consumer.Close()

	log.Println("Starting Kafka Consumer...")
	consumer.Consume(context.Background())
}
