//go:build solderinject

package main

import "github.com/muratmirgun/solder"

// NewServerFor builds a Server.
func NewServerFor(cfg Config) (*Server, error) {
	panic(solder.Build(solder.Provide(NewDB, NewServer)))
}
