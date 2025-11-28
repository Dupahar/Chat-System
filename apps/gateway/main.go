package main

import (
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	f, err := os.OpenFile("gateway.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	kafkaBrokersStr := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokersStr == "" {
		kafkaBrokersStr = "localhost:19092"
	}
	kafkaBrokers := strings.Split(kafkaBrokersStr, ",")

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	kafkaTopic := "chat-messages"

	hub := NewHub(kafkaBrokers, kafkaTopic, redisAddr)
	go hub.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	log.Println("Gateway Service Starting on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
