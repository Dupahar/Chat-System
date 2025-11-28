package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mahaj/networking-minor/pkg/auth"
	"github.com/mahaj/networking-minor/pkg/model"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Client ID (e.g., user ID)
	ID string

	// Channel ID the client is connected to
	ChannelID string
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		// Construct Message struct
		// We assume the client sends raw text content for now.
		// In a future update, client could send JSON with type (e.g. typing).
		// For now, we infer it's a standard message.

		// Try to parse as JSON to see if it has a type, else treat as raw content
		var partialMsg struct {
			Type    model.MessageType `json:"type"`
			Content string            `json:"content"`
		}

		msg := &model.Message{
			ChannelID: c.ChannelID,
			UserID:    c.ID,
			Timestamp: time.Now(),
		}

		if err := json.Unmarshal(message, &partialMsg); err == nil && partialMsg.Type != "" {
			msg.Type = partialMsg.Type
			msg.Content = partialMsg.Content
		} else {
			msg.Type = model.TypeMessage
			msg.Content = string(message)
		}

		c.hub.broadcast <- msg
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// Extract User ID from Auth Token
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		// Try query param as fallback (standard for some WS clients)
		tokenString = r.URL.Query().Get("token")
	}

	if tokenString == "" {
		log.Println("Unauthorized: No token provided")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Remove "Bearer " prefix if present
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	claims, err := auth.ValidateToken(tokenString)
	if err != nil {
		log.Printf("Unauthorized: Invalid token: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := claims.UserID

	// Get Channel ID from query param
	channelID := r.URL.Query().Get("channel")
	if channelID == "" {
		channelID = "general"
	}

	// Validate DM access
	if len(channelID) > 3 && channelID[:3] == "dm:" {
		parts := strings.Split(channelID, ":")
		if len(parts) != 3 {
			http.Error(w, "Invalid DM channel format", http.StatusBadRequest)
			return
		}
		// parts[1] and parts[2] are the user IDs
		if parts[1] != userID && parts[2] != userID {
			http.Error(w, "Unauthorized to join this DM", http.StatusForbidden)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), ID: userID, ChannelID: channelID}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
