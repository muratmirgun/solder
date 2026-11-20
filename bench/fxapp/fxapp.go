// Package fxapp wires bench/app with uber-go/fx, the same graph as solderapp.
package fxapp

import (
	"github.com/muratmirgun/solder/bench/app"
	"go.uber.org/fx"
)

func New(cfg app.Config) *fx.App {
	return fx.New(
		fx.NopLogger,
		fx.Supply(cfg),
		fx.Provide(
			app.NewLogger, app.NewDB, app.NewCache,
			app.NewUserRepo, app.NewOrderRepo, app.NewProductRepo,
			app.NewUserService, app.NewOrderService, app.NewProductService,
			app.NewHandler, app.NewServer,
		),
		fx.Invoke(func(lc fx.Lifecycle, db *app.DB, c *app.Cache, s *app.Server) {
			lc.Append(fx.Hook{OnStart: db.Start, OnStop: db.Stop})
			lc.Append(fx.Hook{OnStart: c.Start, OnStop: c.Stop})
			lc.Append(fx.Hook{OnStart: s.Start, OnStop: s.Stop})
		}),
	)
}
