package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mahaj/networking-minor/pkg/model"
)

type LoginResponse struct {
	Token string `json:"token"`
}

func login(apiAddr, userID string) (string, error) {
	reqBody, _ := json.Marshal(map[string]string{"user_id": userID})
	resp, err := http.Post(apiAddr+"/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed: %s", string(body))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", err
	}

	return loginResp.Token, nil
}

func main() {
	serverAddr := flag.String("addr", "localhost:8080", "gateway service address")
	apiAddr := flag.String("api", "http://localhost:8081", "api service address")
	userID := flag.String("user", "user1", "user id")
	channelID := flag.String("channel", "general", "channel id")
	dmUser := flag.String("dm", "", "user id to dm (overrides -channel)")
	flag.Parse()

	// Determine Channel ID
	finalChannelID := *channelID
	if *dmUser != "" {
		// Sort user IDs to ensure consistent channel ID
		u1, u2 := *userID, *dmUser
		if u1 > u2 {
			u1, u2 = u2, u1
		}
		finalChannelID = fmt.Sprintf("dm:%s:%s", u1, u2)
	}

	// 1. Login to get token
	log.Printf("Logging in as %s...", *userID)
	token, err := login(*apiAddr, *userID)
	if err != nil {
		log.Fatal("Login failed:", err)
	}
	log.Printf("Login successful. Token: %s...", token[:10])

	// 2. Connect to WebSocket with token
	u := url.URL{Scheme: "ws", Host: *serverAddr, Path: "/ws"}
	log.Printf("connecting to %s", u.String())

	header := http.Header{}
	header.Add("Authorization", "Bearer "+token)
	// Pass channel via query param (Gateway will read this)
	q := u.Query()
	q.Set("channel", finalChannelID)
	u.RawQuery = q.Encode()

	c, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	// 3. Start goroutine to read messages
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}

			// Parse JSON message
			var msg model.Message
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("Received raw: %s", message)
				continue
			}

			if msg.Type == model.TypeTyping {
				fmt.Printf("\rUser %s is typing...      \n> ", msg.UserID)
			} else {
				fmt.Printf("\r%s: %s\n> ", msg.UserID, msg.Content)
			}
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// 4. Read from stdin and send messages
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("> ")
		for scanner.Scan() {
			text := scanner.Text()
			if text == "" {
				fmt.Print("> ")
				continue
			}

			if text == "/quit" {
				close(interrupt)
				break
			}

			if text == "/typing" {
				// Send typing event
				msg := model.Message{
					Type: model.TypeTyping,
				}
				jsonMsg, _ := json.Marshal(msg)
				err := c.WriteMessage(websocket.TextMessage, jsonMsg)
				if err != nil {
					log.Println("write:", err)
					break
				}
				fmt.Print("> ")
				continue
			}

			// Send normal message
			// Client sends raw text, Gateway wraps it.
			err := c.WriteMessage(websocket.TextMessage, []byte(text))
			if err != nil {
				log.Println("write:", err)
				break
			}
			fmt.Print("> ")
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
