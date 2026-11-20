package main

import (
	"fmt"

	"github.com/muratmirgun/solder"
)

type DB struct{ name string }

type DBs struct {
	solder.Out
	Primary *DB `name:"primary"`
	Replica *DB `name:"replica"`
}

func NewDBs() DBs { return DBs{Primary: &DB{"primary"}, Replica: &DB{"replica"}} }

type Route string

type RouteOut struct {
	solder.Out
	Route Route `group:"routes"`
}

func Health() RouteOut { return RouteOut{Route: "/health"} }
func Users() RouteOut  { return RouteOut{Route: "/users"} }

type Params struct {
	solder.In
	Write  *DB     `name:"primary"`
	Read   *DB     `name:"replica"`
	Routes []Route `group:"routes"`
}

type Server struct{ p Params }

func NewServer(p Params) *Server { return &Server{p} }

func main() {
	s := NewServerApp()
	fmt.Println(s.p.Write.name, s.p.Read.name, s.p.Routes)
}
