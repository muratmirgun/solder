//go:build solderinject

package main

import "github.com/muratmirgun/solder"

func Inj() A { panic(solder.Build(solder.Supply(5))) }
