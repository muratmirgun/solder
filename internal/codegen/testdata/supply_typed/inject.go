//go:build solderinject

package main

import (
	"time"

	"github.com/muratmirgun/solder"
)

func NewPoolApp() *Pool {
	panic(solder.Build(solder.Supply(maxConns, 3*time.Second), solder.Provide(NewPool)))
}
