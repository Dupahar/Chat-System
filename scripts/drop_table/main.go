package main

import (
	"log"

	"github.com/mahaj/networking-minor/pkg/db"
)

func main() {
	scyllaHosts := []string{"localhost:9042"}
	keyspace := "chat"

	session, err := db.NewSession(scyllaHosts, keyspace)
	if err != nil {
		log.Fatalf("Failed to connect to ScyllaDB: %v", err)
	}
	defer session.Close()

	log.Println("Dropping table messages...")
	if err := session.Query("DROP TABLE IF EXISTS messages").Exec(); err != nil {
		log.Fatalf("Failed to drop table: %v", err)
	}
	log.Println("Table dropped successfully.")
}
