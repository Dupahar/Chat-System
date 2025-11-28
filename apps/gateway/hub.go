package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/mahaj/networking-minor/pkg/model"
	"github.com/mahaj/networking-minor/pkg/snowflake"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

type Hub struct {
	clients     map[string]map[*Client]bool // channel_id -> clients
	userClients map[string]map[*Client]bool // user_id -> clients (Global tracking)
	broadcast   chan *model.Message
	register    chan *Client
	unregister  chan *Client
	mu          sync.RWMutex
	producer    *kafka.Writer
	redis       *redis.Client
	snowflake   *snowflake.Node
}

func NewHub(kafkaBrokers []string, topic string, redisAddr string) *Hub {
	producer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBrokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Consumer for fanout
	consumer := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     kafkaBrokers,
		Topic:       topic,
		GroupID:     "gateway-group-" + time.Now().String(), // Unique group for fanout (broadcast to all gateways)
		StartOffset: kafka.LastOffset,
		MinBytes:    10e3,
		MaxBytes:    10e6,
	})

	// Initialize Snowflake Node
	// In production, node ID should be unique per instance (e.g., from env var or service discovery)
	node, err := snowflake.NewNode(1)
	if err != nil {
		log.Fatalf("Failed to initialize snowflake node: %v", err)
	}

	h := &Hub{
		broadcast:   make(chan *model.Message),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		clients:     make(map[string]map[*Client]bool),
		userClients: make(map[string]map[*Client]bool),
		producer:    producer,
		redis:       rdb,
		snowflake:   node,
	}

	// Start consumer
	go func() {
		defer consumer.Close()
		for {
			m, err := consumer.ReadMessage(context.Background())
			if err != nil {
				log.Printf("Gateway consumer error: %v", err)
				break
			}

			// Parse JSON message to get channel_id
			var msg model.Message
			if err := json.Unmarshal(m.Value, &msg); err != nil {
				log.Printf("Failed to unmarshal message from Kafka: %v", err)
				continue
			}

			h.mu.RLock()
			// DM Routing: If channel starts with "dm:", route to participants globally
			if len(msg.ChannelID) > 3 && msg.ChannelID[:3] == "dm:" {
				parts := strings.Split(msg.ChannelID, ":")
				if len(parts) == 3 {
					// parts[1] and parts[2] are user IDs
					recipients := []string{parts[1], parts[2]}
					for _, userID := range recipients {
						if clients, ok := h.userClients[userID]; ok {
							for client := range clients {
								select {
								case client.send <- m.Value:
								default:
									close(client.send)
									delete(clients, client)
								}
							}
						}
					}
				}
			} else {
				// Standard Channel Routing
				if clients, ok := h.clients[msg.ChannelID]; ok {
					for client := range clients {
						select {
						case client.send <- m.Value:
						default:
							close(client.send)
							delete(clients, client)
						}
					}
				}
			}
			h.mu.RUnlock()
		}
	}()

	return h
}

func (h *Hub) Run() {
	defer h.producer.Close()
	defer h.redis.Close()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// Register in Channel Map
			if h.clients[client.ChannelID] == nil {
				h.clients[client.ChannelID] = make(map[*Client]bool)
			}
			h.clients[client.ChannelID][client] = true

			// Register in Global User Map
			if h.userClients[client.ID] == nil {
				h.userClients[client.ID] = make(map[*Client]bool)
			}
			h.userClients[client.ID][client] = true
			h.mu.Unlock()

			// Set presence in Redis Set
			err := h.redis.SAdd(context.Background(), "channel:"+client.ChannelID+":users", client.ID).Err()
			if err != nil {
				log.Printf("Failed to set presence for %s: %v", client.ID, err)
			}
			log.Printf("Client registered: %s in channel %s", client.ID, client.ChannelID)

			// Broadcast Join Event
			go func() {
				h.broadcast <- &model.Message{
					ChannelID: client.ChannelID,
					UserID:    client.ID,
					Type:      model.TypePresence,
					Content:   "joined",
					Timestamp: time.Now(),
				}
			}()

		case client := <-h.unregister:
			h.mu.Lock()
			// Unregister from Channel Map
			if clients, ok := h.clients[client.ChannelID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.ChannelID)
					}

					// Remove presence from Redis Set
					err := h.redis.SRem(context.Background(), "channel:"+client.ChannelID+":users", client.ID).Err()
					if err != nil {
						log.Printf("Failed to delete presence for %s: %v", client.ID, err)
					}
					log.Printf("Client unregistered: %s from channel %s", client.ID, client.ChannelID)

					// Broadcast Leave Event
					go func() {
						h.broadcast <- &model.Message{
							ChannelID: client.ChannelID,
							UserID:    client.ID,
							Type:      model.TypePresence,
							Content:   "left",
							Timestamp: time.Now(),
						}
					}()
				}
			}

			// Unregister from Global User Map
			if clients, ok := h.userClients[client.ID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					if len(clients) == 0 {
						delete(h.userClients, client.ID)
					}
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			// Assign ID and Timestamp if not present
			if msg.ID == 0 {
				msg.ID = h.snowflake.Generate()
			}
			if msg.Timestamp.IsZero() {
				msg.Timestamp = time.Now()
			}

			// Marshal to JSON
			jsonMsg, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				continue
			}

			// Publish to Kafka
			err = h.producer.WriteMessages(context.Background(),
				kafka.Message{
					Value: jsonMsg,
					Time:  time.Now(),
				},
			)
			if err != nil {
				log.Printf("Failed to write message to Kafka: %v", err)
			} else {
				log.Printf("Message published to Kafka: %s", string(jsonMsg))
			}
		}
	}
}
