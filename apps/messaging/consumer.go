package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/mahaj/networking-minor/pkg/db"
	"github.com/mahaj/networking-minor/pkg/model"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
	db     *db.Session
}

func NewConsumer(brokers []string, topic string, groupID string, session *db.Session) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	return &Consumer{reader: r, db: session}
}

func (c *Consumer) Consume(ctx context.Context) {
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("Error reading message: %v. Retrying in 1s...", err)
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("Received message from Kafka: %s", string(m.Value))

		// Parse JSON message
		var msg model.Message
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		// Only persist actual messages
		if msg.Type != model.TypeMessage {
			log.Printf("Skipping persistence for ephemeral message type: %s", msg.Type)
			continue
		}

		// Persist to ScyllaDB
		query := `INSERT INTO messages (channel_id, id, user_id, content, timestamp) VALUES (?, ?, ?, ?, ?)`

		if err := c.db.Query(query, msg.ChannelID, msg.ID, msg.UserID, msg.Content, msg.Timestamp).Exec(); err != nil {
			log.Printf("Failed to save message to ScyllaDB: %v", err)
		} else {
			log.Printf("Message saved to ScyllaDB: %d", msg.ID)
		}

		// DM Persistence: Update user_conversations table
		if len(msg.ChannelID) > 3 && msg.ChannelID[:3] == "dm:" {
			parts := strings.Split(msg.ChannelID, ":")
			if len(parts) == 3 {
				u1 := parts[1]
				u2 := parts[2]

				// Insert for u1 (u1 talks to u2)
				q1 := `INSERT INTO user_conversations (user_id, other_user_id, last_updated) VALUES (?, ?, ?)`
				if err := c.db.Query(q1, u1, u2, msg.Timestamp).Exec(); err != nil {
					log.Printf("Failed to update conversation for %s: %v", u1, err)
				}

				// Insert for u2 (u2 talks to u1)
				q2 := `INSERT INTO user_conversations (user_id, other_user_id, last_updated) VALUES (?, ?, ?)`
				if err := c.db.Query(q2, u2, u1, msg.Timestamp).Exec(); err != nil {
					log.Printf("Failed to update conversation for %s: %v", u2, err)
				}

				// Increment unread count for the recipient
				// If u1 sent to u2, then u2 has an unread message from u1
				// We need to know who sent it. msg.UserID is the sender.
				sender := msg.UserID
				var recipient string
				if u1 == sender {
					recipient = u2
				} else {
					recipient = u1
				}

				qCounter := `UPDATE conversation_counters SET unread_count = unread_count + 1 WHERE user_id = ? AND other_user_id = ?`
				if err := c.db.Query(qCounter, recipient, sender).Exec(); err != nil {
					log.Printf("Failed to increment unread count for %s: %v", recipient, err)
				}
			}
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
