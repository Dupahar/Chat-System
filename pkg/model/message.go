package model

import "time"

type MessageType string

const (
	TypeMessage     MessageType = "message"
	TypeTyping      MessageType = "typing"
	TypePresence    MessageType = "presence"
	TypeReadReceipt MessageType = "read_receipt"
)

type Message struct {
	ID        int64       `json:"id"`
	ChannelID string      `json:"channel_id"`
	UserID    string      `json:"user_id"`
	Content   string      `json:"content"`
	Type      MessageType `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
}
