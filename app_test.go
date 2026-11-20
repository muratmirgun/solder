package solder

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// recorder builds hooks that log their calls and can be told to fail.
type recorder struct{ calls []string }

func (r *recorder) hook(name string, failStart, failStop bool) Hook {
	return Hook{
		OnStart: func(context.Context) error {
			r.calls = append(r.calls, "start "+name)
			if failStart {
				return errors.New("start " + name + " failed")
			}
			return nil
		},
		OnStop: func(context.Context) error {
			r.calls = append(r.calls, "stop "+name)
			if failStop {
				return errors.New("stop " + name + " failed")
			}
			return nil
		},
	}
}

func (r *recorder) want(t *testing.T, want ...string) {
	t.Helper()
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls = %q, want %q", r.calls, want)
	}
}

func TestStartStopOrder(t *testing.T) {
	var r recorder
	app := New()
	app.Append(r.hook("a", false, false))
	app.Append(Hook{}) // nil callbacks are skipped
	app.Append(r.hook("b", false, false))
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := app.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if err := app.Stop(ctx); err != nil { // second Stop is a no-op
		t.Fatal(err)
	}
	r.want(t, "start a", "start b", "stop b", "stop a")
}

func TestStartFailureRollsBack(t *testing.T) {
	var r recorder
	app := New()
	app.Append(r.hook("a", false, true))
	app.Append(r.hook("b", true, false))
	app.Append(r.hook("c", false, false))
	err := app.Start(context.Background())
	if err == nil || err.Error() != "start b failed\nstop a failed" {
		t.Fatalf("err = %v", err)
	}
	// b never started, so it is not stopped; c never ran.
	r.want(t, "start a", "start b", "stop a")
}

func TestStopKeepsGoingOnError(t *testing.T) {
	var r recorder
	app := New()
	app.Append(r.hook("a", false, true))
	app.Append(r.hook("b", false, true))
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	err := app.Stop(ctx)
	if err == nil || err.Error() != "stop b failed\nstop a failed" {
		t.Fatalf("err = %v", err)
	}
	r.want(t, "start a", "start b", "stop b", "stop a")
}

func TestRunStopsAfterShutdownSignal(t *testing.T) {
	var r recorder
	app := New()
	app.Append(r.hook("a", false, false))
	ctx, cancel := context.WithCancel(context.Background())
	app.Append(Hook{OnStart: func(context.Context) error { cancel(); return nil }}) // "signal" once started
	if err := app.run(ctx); err != nil {
		t.Fatal(err)
	}
	r.want(t, "start a", "stop a")
}

func TestRollbackGetsLiveContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var stopErr error
	app := New()
	app.Append(Hook{OnStop: func(ctx context.Context) error { stopErr = ctx.Err(); return nil }})
	app.Append(Hook{OnStart: func(context.Context) error { cancel(); return context.Canceled }})
	if err := app.Start(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if stopErr != nil {
		t.Fatalf("rollback ran with a dead context: %v", stopErr)
	}
}

// ADR 0005: Start and Stop allocate nothing once hooks are registered.
func TestStartStopZeroAlloc(t *testing.T) {
	noop := func(context.Context) error { return nil }
	app := New()
	for range 8 {
		app.Append(Hook{OnStart: noop, OnStop: noop})
	}
	ctx := context.Background()
	allocs := testing.AllocsPerRun(100, func() {
		_ = app.Start(ctx)
		_ = app.Stop(ctx)
	})
	if allocs != 0 {
		t.Fatalf("Start+Stop allocated %v times per run, want 0", allocs)
	}
}

func BenchmarkStartStop(b *testing.B) {
	noop := func(context.Context) error { return nil }
	app := New()
	for range 8 {
		app.Append(Hook{OnStart: noop, OnStop: noop})
	}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		_ = app.Start(ctx)
		_ = app.Stop(ctx)
	}
}
