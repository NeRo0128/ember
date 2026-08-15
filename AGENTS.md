# AGENTS.md

Go logging library. Module path is `github.com/NeRo0128/logcore` — imports look like `github.com/NeRo0128/logcore/logger` and `github.com/NeRo0128/logcore/internal/utils`. Go 1.24, toolchain pinned in `go.mod`.

## Layout

- `logger/` — the public library: `logger.go` (interface + impl), `options.go` (functional options), `logger_test.go`.
- `internal/utils/formatter.go` — all JSON/text formatting and color logic. Only reachable from within this module (Go `internal/` rule); keep format logic here, not in `logger`.
- `logs.go` — root package `logcore`, a thin global wrapper over a package-level `defaultLogger` (`LogInfo`, `LogError`, etc.).

## Commands

```sh
go build ./...      # no Makefile in repo
go vet ./...        # also run gofmt
go test ./...       # or: go test ./logger/
go test -race ./... # CI runs this; catch data races before pushing
```

Tests use `github.com/stretchr/testify/assert` and `bytes.Buffer` via `AddOutput`. Extend `logger/logger_test.go`, not `logs.go`.

## Gotchas (verified from source)

- `logger.Fatal` only logs at `FatalLevel` — it does **not** `os.Exit`. Callers that expect termination must do it themselves.
- `NewLogger` defaults: `InfoLevel`, text output, single `os.Stdout` writer, caller off, no layer. `WithJSON(true)` + `WithPrettyJSON(true)` for pretty JSON.
- Levels are ordered `Debug(0) < Info < Warn < Error < Fatal(4)`; messages below the configured level are dropped (`l.level > level`).
- Concurrency: `loggerImpl` is mutex-protected; `WithFields`/`WithLayer`/`WithContext` return clones, but `WithField` (option), `SetLevel`, and `AddOutput` mutate the shared instance. Use the `With*` methods to derive loggers without affecting the original.
- Colors in text output depend on **`os.Stdout` being a TTY** (`term.IsTerminal(os.Stdout.Fd())` in `utils`), not on the writer being written to — buffer-based tests always get plain text.
- `FormatStructAsJSON` writes raw indented JSON straight to writers, bypassing level filtering and the standard `ts`/`lvl`/`msg` entry shape.

## Conventions

- Code comments, README, and commit messages are in Spanish — keep new comments consistent.
- `runtime.Caller(2)` comment ("skip 3 frames") is stale relative to the code; caller frame counts are fragile, verify before editing.
