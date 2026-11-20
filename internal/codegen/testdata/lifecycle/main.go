package main

import (
	"context"
	"fmt"

	"github.com/muratmirgun/solder"
)

type Logger struct{}

func (*Logger) Log(s string) { fmt.Println(s) }

func NewLogger(lc solder.Lifecycle) *Logger {
	l := &Logger{}
	lc.Append(solder.Hook{
		OnStart: func(context.Context) error { l.Log("logger start"); return nil },
		OnStop:  func(context.Context) error { l.Log("logger stop"); return nil },
	})
	return l
}

type Server struct{ log *Logger }

func NewServer(l *Logger) *Server             { return &Server{log: l} }
func (s *Server) Start(context.Context) error { s.log.Log("server start"); return nil }
func (s *Server) Stop(context.Context) error  { s.log.Log("server stop"); return nil }
func Announce(l *Logger) error                { l.Log("built"); return nil }

func main() {
	app, err := NewApp()
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		panic(err)
	}
	if err := app.Stop(ctx); err != nil {
		panic(err)
	}
}
