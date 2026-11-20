package greet

import "github.com/muratmirgun/solder"

type Greeter interface{ Greet() string }

type English struct{ Name string }

func (e English) Greet() string      { return "hello " + e.Name }
func NewEnglish(name string) English { return English{Name: name} }

var Module = solder.Module("greet",
	solder.Provide(NewEnglish),
	solder.Bind[Greeter, English](),
)
