# Contributing to solder

solder is under construction and the API is still moving, so the most useful
contributions right now are issues and design feedback.

- **Bugs:** open an issue with `go version`, the command you ran, the
  injector file and the full output.
- **Features / API changes:** open an issue first. Design decisions are
  written as ADRs in `docs/adr/` before code.

## Pull requests

1. Agree on the approach in an issue.
2. Generator changes come with a golden case in
   `internal/codegen/testdata/<case>/` (`want_gen.go` + `want_out.txt`, or
   `want_err.txt`). Refresh goldens with
   `go test ./internal/codegen -update` only when the change is intended.
3. `gofmt -s -l .`, `go vet ./...` and `go test -race ./...` must be clean.
4. Runtime changes must keep `TestStartStopZeroAlloc` passing.

Contributions are licensed under Apache 2.0 (see `LICENSE`).
