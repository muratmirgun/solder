// Command solder-app starts and stops the bench graph with solder.
package main

import (
	"context"
	"log"

	"github.com/muratmirgun/solder/bench/app"
	"github.com/muratmirgun/solder/bench/solderapp"
)

func main() {
	a, err := solderapp.New(app.Config{Addr: ":8080", DSN: "pg://"})
	if err != nil {
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
