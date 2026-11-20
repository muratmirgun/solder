//go:build solderinject

package solderapp

import (
	"github.com/muratmirgun/solder"
	"github.com/muratmirgun/solder/bench/app"
)

func New(cfg app.Config) (*solder.App, error) {
	panic(solder.Build(
		solder.Provide(
			app.NewLogger,
			app.NewUserRepo, app.NewOrderRepo, app.NewProductRepo,
			app.NewUserService, app.NewOrderService, app.NewProductService,
			app.NewHandler,
		),
		solder.Lifecycled(app.NewDB, app.NewCache, app.NewServer),
	))
}
