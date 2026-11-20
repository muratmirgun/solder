package solder

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Hook is a pair of callbacks run when an App starts and stops.
// Either field may be nil.
type Hook struct {
	OnStart func(context.Context) error
	OnStop  func(context.Context) error
}

// Lifecycle is what providers receive to register hooks.
// Generated code passes the *App being built.
type Lifecycle interface {
	Append(Hook)
}

// App runs the hooks registered while the object graph was built.
// It is not safe for concurrent use: hooks are appended during construction,
// then Start and Stop are called from one goroutine.
type App struct {
	hooks   []Hook
	started int // hooks[:started] have run OnStart successfully
}

// New returns an empty App. Generated injectors call it.
func New() *App { return &App{} }

var _ Lifecycle = (*App)(nil)

// Append registers h. Hooks start in the order they were appended and stop in
// reverse.
func (a *App) Append(h Hook) { a.hooks = append(a.hooks, h) }

// Start runs every OnStart in order. If one fails, the hooks that already
// started are stopped in reverse and both errors are returned.
func (a *App) Start(ctx context.Context) error {
	for a.started < len(a.hooks) {
		if f := a.hooks[a.started].OnStart; f != nil {
			if err := f(ctx); err != nil {
				// ctx may be why start failed (deadline), so the rollback gets its own.
				stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), hookTimeout)
				defer cancel()
				return errors.Join(err, a.Stop(stopCtx))
			}
		}
		a.started++
	}
	return nil
}

// Stop runs OnStop for every started hook in reverse order. It keeps going
// when a hook fails and returns all errors. Calling it again is a no-op.
func (a *App) Stop(ctx context.Context) error {
	var errs []error
	for ; a.started > 0; a.started-- {
		if f := a.hooks[a.started-1].OnStop; f != nil {
			if err := f(ctx); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// shortcut: fixed timeouts (fx's default); make them configurable when someone needs it.
const hookTimeout = 15 * time.Second

// Run starts the app, waits for SIGINT or SIGTERM, then stops it.
func (a *App) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return a.run(ctx)
}

// run is Run with the shutdown signal supplied as ctx, so tests need no signals.
func (a *App) run(ctx context.Context) error {
	startCtx, cancel := context.WithTimeout(ctx, hookTimeout)
	err := a.Start(startCtx)
	cancel()
	if err != nil {
		return err
	}
	<-ctx.Done()
	// ctx is already done, so stopping gets a fresh deadline.
	stopCtx, cancel := context.WithTimeout(context.Background(), hookTimeout)
	defer cancel()
	return a.Stop(stopCtx)
}
