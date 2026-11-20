// Command fx-app starts and stops the bench graph with fx.
package main

import (
	"context"
	"log"

	"github.com/muratmirgun/solder/bench/app"
	"github.com/muratmirgun/solder/bench/fxapp"
)

func main() {
	a := fxapp.New(app.Config{Addr: ":8080", DSN: "pg://"})
	if err := a.Err(); err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	if err := a.Start(ctx); err != nil {
		log.Fatal(err)
	}
	if err := a.Stop(ctx); err != nil {
		log.Fatal(err)
	}
}
