package main

import (
	"fmt"
	"time"
)

const maxConns int = 4

type Pool struct {
	size    int
	timeout time.Duration
}

func NewPool(size int, timeout time.Duration) *Pool { return &Pool{size, timeout} }

func main() { fmt.Println(*NewPoolApp()) }
