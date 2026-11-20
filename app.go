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
	svc     any // from AppendService: its methods are called directly, so no closure is allocated
}

type starter interface{ Start(context.Context) error }
type stopper interface{ Stop(context.Context) error }

func (h *Hook) start(ctx context.Context) error {
	if h.OnStart != nil {
		return h.OnStart(ctx)
	}
	if s, ok := h.svc.(starter); ok {
		return s.Start(ctx)
	}
	return nil
}

func (h *Hook) stop(ctx context.Context) error {
	if h.OnStop != nil {
		return h.OnStop(ctx)
	}
	if s, ok := h.svc.(stopper); ok {
		return s.Stop(ctx)
	}
	return nil
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
	started int     // hooks[:started] have run OnStart successfully
	buf     [8]Hook // backing array for the first hooks, so typical apps allocate only the App
}

// New returns an empty App. Generated injectors call it.
func New() *App {
	a := &App{}
	a.hooks = a.buf[:0]
	return a
}

var _ Lifecycle = (*App)(nil)

// Append registers h. Hooks start in the order they were appended and stop in
// reverse.
func (a *App) Append(h Hook) { a.hooks = append(a.hooks, h) }

// AppendService registers svc's Start(context.Context) error and
// Stop(context.Context) error methods, whichever it has, as a hook. Unlike
// Append(Hook{OnStart: svc.Start}) it does not allocate. Generated code uses it
// for solder.Lifecycled.
func (a *App) AppendService(svc any) { a.hooks = append(a.hooks, Hook{svc: svc}) }

// Start runs every OnStart in order. If one fails, the hooks that already
// started are stopped in reverse and both errors are returned.
func (a *App) Start(ctx context.Context) error {
	for a.started < len(a.hooks) {
		if err := a.hooks[a.started].start(ctx); err != nil {
			// ctx may be why start failed (deadline), so the rollback gets its own.
			stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), hookTimeout)
			defer cancel()
			return errors.Join(err, a.Stop(stopCtx))
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
		if err := a.hooks[a.started-1].stop(ctx); err != nil {
			errs = append(errs, err)
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
