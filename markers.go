// Package solder is compile-time dependency injection for Go.
//
// You describe the object graph in an injector function, in a file with the
// solderinject build tag:
//
//	//go:build solderinject
//
//	func NewApp() (*solder.App, error) {
//		panic(solder.Build(
//			solder.Provide(NewConfig, NewDB),
//			solder.Lifecycled(NewServer),
//		))
//	}
//
// Running the solder command writes solder_gen.go with plain Go that builds the
// graph: direct calls, no reflection. The marker functions in this file are
// read by the generator. They do nothing at runtime, so module variables such
// as `var Module = solder.Module(...)` are safe in normal files; only Build
// panics, because a Build that runs means solder gen was not run.
//
// See docs/adr in the repository for the design.
package solder

// Option is one entry passed to Build or Module.
type Option struct{}

// marker is what every option returns; options only matter to the generator.
func marker() Option { return Option{} }

// Build marks a function as an injector. Its body must be exactly
// panic(solder.Build(...)).
func Build(...Option) string { panic("solder: injector called before `solder gen` replaced it") }

// Provide adds constructors. Each must be a package-level function returning
// T or (T, error).
func Provide(...any) Option { return marker() }

// Lifecycled is Provide, plus registering the value's
// Start(context.Context) error and Stop(context.Context) error methods as a
// Hook. Lifecycled values are always constructed.
func Lifecycled(...any) Option { return marker() }

// Module groups options under a name. Decorate inside a module only affects
// that module and its children.
func Module(name string, opts ...Option) Option { return marker() }

// Supply adds values used as-is, keyed by their static type. Untyped
// constants are rejected; convert them first.
func Supply(...any) Option { return marker() }

// Bind makes the provided T available as the interface I.
func Bind[I, T any]() Option { return marker() }

// Invoke calls functions during construction, in order. Each may return an
// error.
func Invoke(...any) Option { return marker() }

// Decorate adds functions of the form func(T, deps...) T or (T, error) that
// wrap T for consumers in the same module and its children.
func Decorate(...any) Option { return marker() }

// In marks a parameter struct: each field is a dependency, optionally tagged
// `name:"x"` or `group:"x"` (the field type is then []T).
type In struct{}

// Out marks a result struct: each field is provided, optionally tagged
// `name:"x"` or `group:"x"`.
type Out struct{}
