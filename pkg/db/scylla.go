package db

import (
	"log"
	"time"

	"github.com/gocql/gocql"
)

type Session struct {
	*gocql.Session
}

func NewSession(hosts []string, keyspace string) (*Session, error) {
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 5 * time.Second
	cluster.ConnectTimeout = 5 * time.Second

	// Retry policy
	cluster.RetryPolicy = &gocql.ExponentialBackoffRetryPolicy{
		NumRetries: 3,
		Min:        100 * time.Millisecond,
		Max:        1 * time.Second,
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	log.Println("Connected to ScyllaDB cluster")
	return &Session{Session: session}, nil
}
