package main

import (
	"errors"
	"fmt"
)

type Config struct{ DSN string }
type DB struct{ dsn string }
type Server struct{ db *DB }

func NewDB(c Config) (*DB, error) {
	if c.DSN == "" {
		return nil, errors.New("empty dsn")
	}
	return &DB{dsn: c.DSN}, nil
}

func NewServer(db *DB) *Server { return &Server{db: db} }

func main() {
	s, err := NewServerFor(Config{DSN: "pg://x"})
	fmt.Println(s.db.dsn, err)
	_, err = NewServerFor(Config{})
	fmt.Println(err)
}
