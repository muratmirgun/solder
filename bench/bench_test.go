// Startup benchmarks: build the bench/app graph (12 components, 3 lifecycle
// hooks), start it and stop it. Run: go test -bench . -benchmem
package bench

import (
	"context"
	"testing"

	"github.com/muratmirgun/solder/bench/app"
	"github.com/muratmirgun/solder/bench/fxapp"
	"github.com/muratmirgun/solder/bench/solderapp"
)

var cfg = app.Config{Addr: ":8080", DSN: "pg://"}

func BenchmarkSolder(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		a, err := solderapp.New(cfg)
		if err != nil {
			b.Fatal(err)
		}
		if err := a.Start(ctx); err != nil {
			b.Fatal(err)
		}
		if err := a.Stop(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFx(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		a := fxapp.New(cfg)
		if err := a.Err(); err != nil {
			b.Fatal(err)
		}
		if err := a.Start(ctx); err != nil {
			b.Fatal(err)
		}
		if err := a.Stop(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
