//go:build solderinject

package main

import (
	"example.com/app/greet"
	"github.com/muratmirgun/solder"
)

func NewPairApp(name string) Pair {
	panic(solder.Build(
		greet.Module,
		solder.Supply(Prefix("> ")),
		solder.Provide(Shout, NewPair),
		solder.Module("loud", solder.Decorate(Loud), solder.Provide(NewInner)),
	))
}
