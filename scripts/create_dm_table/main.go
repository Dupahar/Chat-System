package main

import (
	"log"

	"github.com/gocql/gocql"
)

func main() {
	cluster := gocql.NewCluster("localhost")
	cluster.Keyspace = "chat"
	cluster.Consistency = gocql.Quorum
	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	err = session.Query(`
		CREATE TABLE IF NOT EXISTS user_conversations (
			user_id text,
			other_user_id text,
			last_updated timestamp,
			PRIMARY KEY (user_id, other_user_id)
		)
	`).Exec()

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Table user_conversations created successfully")
}
