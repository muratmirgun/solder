//go:build solderinject

package main

import "github.com/muratmirgun/solder"

func Inj() (A, error) { panic(solder.Build(solder.Provide(NewAErr), solder.Supply(A{}))) }
