//go:build solderinject

package main

import "github.com/muratmirgun/solder"

func NewServerApp() *Server {
	panic(solder.Build(solder.Provide(NewDBs, Health, Users, NewServer)))
}
