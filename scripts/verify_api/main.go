package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type LoginResponse struct {
	Token string `json:"token"`
}

func main() {
	apiAddr := "http://localhost:8081"

	// 1. Login
	reqBody, _ := json.Marshal(map[string]string{"user_id": "test_user"})
	resp, err := http.Post(apiAddr+"/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Token: %s...\n", loginResp.Token[:10])

	// 2. Get History for DM
	log.Println("Fetching history for dm:userA:userB...")
	req, _ := http.NewRequest("GET", apiAddr+"/history?channel_id=dm:userA:userB", nil)
	req.Header.Add("Authorization", "Bearer "+loginResp.Token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("History request failed:", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("History: %s", string(body))
}
