<div align="center">

# 🔥 ember

**High-performance structured logging for Go**

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)](https://golang.org)
[![Go Report Card](https://goreportcard.com/badge/github.com/NeRo0128/ember)](https://goreportcard.com/report/github.com/NeRo0128/ember)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

</div>

`ember` is a lightweight, zero-allocation-oriented structured logger for Go, designed for production services that demand **speed**, **safety**, and **observability**.

It provides leveled logging with structured fields, JSON/text output, concurrent-safe writes, and production-grade features — sampling, hooks, async mode, file rotation, and context extractors — without the bloat of larger frameworks.

---

## 📋 Table of Contents

- [Features](#-features)
- [Installation](#-installation)
- [Quick Start](#-quick-start)
- [Production Patterns](#-production-patterns)
  - [Async Logging](#async-logging)
  - [Sampling](#sampling)
  - [Hooks](#hooks)
  - [File Rotation](#file-rotation)
  - [Distributed Tracing](#distributed-tracing)
- [API Reference](#-api-reference)
- [Benchmarks](#-benchmarks)
- [Testing](#-testing)
- [License](#-license)

---

## ✨ Features

| Feature                    | Description                                                                               |
| -------------------------- | ----------------------------------------------------------------------------------------- |
| **Structured Logging**     | JSON or colored plain-text output with typed `Field` key-value pairs                      |
| **Leveled Output**         | `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL` with atomic, lock-free level filtering          |
| **Concurrent-Safe**        | `sync/atomic` for level checks; mutex-protected writes prevent interleaved output         |
| **Sampling**               | `EveryNSampler` reduces log volume for high-throughput services; errors are never sampled |
| **Hooks**                  | Intercept entries for Slack alerts, Prometheus metrics, SIEM forwarding, or audit trails  |
| **Async Mode**             | Buffered channel + worker goroutine keeps your hot path non-blocking                      |
| **File Rotation**          | Automatic rotation by size, backup count, and age with zero external dependencies         |
| **Context Extractors**     | Auto-inject `trace_id`, `span_id`, `user_id` from `context.Context` into every entry      |
| **Zero Core Dependencies** | Only `stretchr/testify` for tests; no runtime bloat                                       |

---

## 📦 Installation

```bash
go get github.com/NeRo0128/ember
```

Requires **Go 1.21+**.

---

## 🚀 Quick Start

```go
package main

import (
    "bytes"
    "fmt"

    "github.com/NeRo0128/ember/logger"
)

func main() {
    log := logger.NewLogger(
        logger.WithLevel(logger.InfoLevel),
        logger.WithJSON(true),
    )

    var buf bytes.Buffer
    log.AddOutput(&buf)

    log.Info("server started",
        logger.Field{Key: "port", Value: 8080},
        logger.Field{Key: "env", Value: "production"},
    )

    fmt.Println(buf.String())
    // Output:
    // {"ts":"2026-08-15T20:00:00Z","lvl":"INFO","msg":"server started","port":8080,"env":"production"}
}
```

### Global Wrapper

For scripts or small utilities, use the package-level convenience functions:

```go
import "github.com/NeRo0128/ember"

ember.LogInfo("application ready")
ember.LogError("connection timeout")
```

---

## 🏭 Production Patterns

### Async Logging

Prevent disk I/O from blocking your HTTP handlers:

```go
log := logger.NewLogger(
    logger.WithLevel(logger.InfoLevel),
    logger.WithAsync(1000), // buffered channel capacity
)

// Non-blocking — returns immediately
log.Info("request handled", logger.Field{Key: "duration_ms", Value: 45})

defer log.Close() // drain queue and flush on shutdown
```

> **Note:** `Fatal` logs are always written synchronously to guarantee program termination.

### Sampling

Reduce noise in high-traffic services while preserving every error:

```go
sampler := logger.NewEveryNSampler(100, logger.InfoLevel)
// 1 of every 100 DEBUG/INFO messages. WARN+ always passes.

log := logger.NewLogger(
    logger.WithLevel(logger.DebugLevel),
    logger.WithSampling(sampler),
)
```

### Hooks

Send errors to Slack, increment Prometheus counters, or forward to a SIEM:

```go
type slackHook struct{}

func (h *slackHook) Levels() []logger.Level {
    return []logger.Level{logger.ErrorLevel, logger.FatalLevel}
}

func (h *slackHook) Fire(e logger.Entry) error {
    // e.Message, e.Level, e.Fields, e.Time, e.Caller, e.Layer
    // send to Slack webhook...
    return nil
}

log
```
