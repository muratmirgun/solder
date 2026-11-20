//go:build solderinject

package main

import "github.com/muratmirgun/solder"

func NewApp() (*solder.App, error) {
	panic(solder.Build(
		solder.Provide(NewLogger),
		solder.Lifecycled(NewServer),
		solder.Invoke(Announce),
	))
}
