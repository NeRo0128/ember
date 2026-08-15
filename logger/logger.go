package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NeRo0128/ember/internal/utils"
)

// Level represents the severity of a log message. Levels are ordered from
// DebugLevel (0) to FatalLevel (4); messages below the logger's configured
// level are discarded.
type Level int32

const (
	// DebugLevel is the lowest severity, for verbose diagnostic output.
	DebugLevel Level = iota
	// InfoLevel reports general application events.
	InfoLevel
	// WarnLevel reports unusual or potentially harmful conditions.
	WarnLevel
	// ErrorLevel reports failures that prevent normal operation.
	ErrorLevel
	// FatalLevel reports unrecoverable errors; logging at this level
	// terminates the program.
	FatalLevel
)

var levelNames = map[Level]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
	FatalLevel: "FATAL",
}

// Logger is the logging API implemented by ember loggers. It offers leveled
// logging with optional structured fields, derived loggers via the With*
// methods, and control over output writers and severity.
//
// A Logger is safe for concurrent use.
type Logger interface {
	// Debug logs a message at DebugLevel with optional fields.
	Debug(msg string, fields ...Field)
	// Info logs a message at InfoLevel with optional fields.
	Info(msg string, fields ...Field)
	// Warn logs a message at WarnLevel with optional fields.
	Warn(msg string, fields ...Field)
	// Error logs a message at ErrorLevel with optional fields.
	Error(msg string, fields ...Field)
	// Fatal logs a message at FatalLevel with optional fields and then
	// terminates the program.
	Fatal(msg string, fields ...Field)

	// FormatStructAsJSON marshals v as indented JSON and writes it raw to
	// every output writer, bypassing level filtering and the standard entry
	// shape.
	FormatStructAsJSON(v interface{})

	// WithFields returns a derived logger carrying the given fields on every
	// entry, in addition to any already set, without modifying the original.
	WithFields(fields ...Field) Logger
	// WithLayer returns a derived logger tagged with the given layer name,
	// without modifying the original.
	WithLayer(layer string) Logger
	// WithContext returns a derived logger bound to ctx, without modifying
	// the original.
	WithContext(ctx context.Context) Logger

	WithExtractors(extractors ...Extractor) Logger
	WithSampling(sampler Sampler) Logger

	// SetLevel changes the minimum severity the logger emits.
	SetLevel(level Level)
	// AddOutput appends a writer to the logger's output destinations.
	AddOutput(w io.Writer)

	AddHook(hook Hook)
	// Sync flushes any output writers that support flushing.
	Sync() error
	// Close closes any output writers implementing io.Closer and clears the
	// output list, preventing writes after close.
	Close() error
}

// Field is a key-value pair attached to a log entry.
type Field struct {
	// Key is the field name used in JSON output and text formatting.
	Key string
	// Value is the field value; any Go value, serialized as JSON when JSON
	// output is enabled.
	Value any
}
type loggerSnapshot struct {
	layer      string
	fields     []Field
	jsonOutput bool
	prettyJSON bool
	out        []io.Writer
	showCaller bool
	hooks      []Hook
	ctx        context.Context
	extractors []Extractor
}

type loggerImpl struct {
	mu         sync.Mutex
	level      atomic.Int32
	layer      string
	fields     []Field
	jsonOutput bool
	prettyJSON bool
	out        []io.Writer
	ctx        context.Context
	showCaller bool

	sampler    Sampler
	hooks      []Hook
	async      bool
	queue      chan func()
	done       chan struct{}
	wg         sync.WaitGroup
	extractors []Extractor
}

var osExit = os.Exit

// NewLogger creates a Logger configured with the given options. It defaults
// to InfoLevel, plain text output, and a single os.Stdout writer. Enable JSON
// output with WithJSON(true) and indentation with WithPrettyJSON(true).
func NewLogger(options ...Option) Logger {
	l := &loggerImpl{
		out:        []io.Writer{os.Stdout},
		jsonOutput: false,
		prettyJSON: false,
	}
	for _, option := range options {
		option(l)
	}
	return l
}

// NewDebugLogger creates a Logger at DebugLevel with the given layer name and
// plain text output to os.Stdout.
func NewDebugLogger(layer string) Logger {
	l := &loggerImpl{
		out:        []io.Writer{os.Stdout},
		jsonOutput: false,
		prettyJSON: false,
		layer:      layer,
	}
	return l
}

// Logger Methods

// SetLevel changes the minimum severity that the logger emits. Messages below
// this level are discarded.
func (l *loggerImpl) SetLevel(level Level) {
	l.level.Store(int32(level))
}

// AddOutput appends an io.Writer to the logger's output destinations. Every
// log entry is written to all registered writers.
func (l *loggerImpl) AddOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = append(l.out, w)
}

func (l *loggerImpl) AddHook(hook Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hooks = append(l.hooks, hook)
}

func (l *loggerImpl) clone() *loggerImpl {
	l.mu.Lock()
	defer l.mu.Unlock()
	return &loggerImpl{
		layer:      l.layer,
		fields:     append([]Field(nil), l.fields...),
		jsonOutput: l.jsonOutput,
		prettyJSON: l.prettyJSON,
		out:        append([]io.Writer(nil), l.out...),
		ctx:        l.ctx,
		showCaller: l.showCaller,
		sampler:    l.sampler,
		hooks:      append([]Hook(nil), l.hooks...),
		async:      l.async,
		extractors: append([]Extractor(nil), l.extractors...),
	}
}

// WithFields returns a derived logger carrying the given fields on every
// entry, in addition to any already set. The original logger is not modified.
func (l *loggerImpl) WithFields(fields ...Field) Logger {
	clone := l.clone()
	clone.fields = append(clone.fields, fields...)
	clone.level.Store(l.level.Load())

	if l.async {
		clone.queue = l.queue
		clone.done = l.done
	}
	return clone
}

// WithContext returns a derived logger bound to ctx. The context can be used
// later to extract values for correlation. The original logger is not modified.
func (l *loggerImpl) WithContext(ctx context.Context) Logger {

	clone := l.clone()
	clone.ctx = ctx
	clone.level.Store(l.level.Load())
	if l.async {
		clone.queue = l.queue
		clone.done = l.done
	}
	return clone
}

func (l *loggerImpl) WithExtractors(extractors ...Extractor) Logger {

	clone := l.clone()
	clone.extractors = append(clone.extractors, extractors...)
	clone.level.Store(l.level.Load())
	if l.async {
		clone.queue = l.queue
		clone.done = l.done
	}
	return clone
}

func (l *loggerImpl) WithSampling(sampler Sampler) Logger {

	clone := l.clone()
	clone.sampler = sampler
	clone.level.Store(l.level.Load())
	if l.async {
		clone.queue = l.queue
		clone.done = l.done
	}
	return clone
}

// WithLayer returns a derived logger tagged with the given layer name, shown
// in text output and JSON entries. The original logger is not modified.
func (l *loggerImpl) WithLayer(layer string) Logger {

	clone := l.clone()
	clone.level.Store(l.level.Load())
	if l.async {
		clone.queue = l.queue
		clone.done = l.done
	}
	return clone
}

// Worker async: consume del canal y ejecuta tareas
func (l *loggerImpl) worker() {
	defer l.wg.Done()
	for {
		select {
		case task, ok := <-l.queue:
			if !ok {
				return
			}
			task()
		case <-l.done:
			// Drain remaining
			for task := range l.queue {
				task()
			}
			return
		}
	}
}

// Writers

// Debug logs a message at DebugLevel with optional fields.
func (l *loggerImpl) Debug(msg string, fields ...Field) {
	if l.level.Load() > int32(DebugLevel) {
		return
	}
	l.log(DebugLevel, msg, fields...)
}

// Info logs a message at InfoLevel with optional fields.
func (l *loggerImpl) Info(msg string, fields ...Field) {
	if l.level.Load() > int32(InfoLevel) {
		return
	}
	l.log(InfoLevel, msg, fields...)
}

// Warn logs a message at WarnLevel with optional fields.
func (l *loggerImpl) Warn(msg string, fields ...Field) {
	if l.level.Load() > int32(WarnLevel) {
		return
	}
	l.log(WarnLevel, msg, fields...)
}

// Error logs a message at ErrorLevel with optional fields.
func (l *loggerImpl) Error(msg string, fields ...Field) {
	if l.level.Load() > int32(ErrorLevel) {
		return
	}
	l.log(ErrorLevel, msg, fields...)
}

// Fatal logs a message at FatalLevel with optional fields and then terminates
// the program via os.Exit.
func (l *loggerImpl) Fatal(msg string, fields ...Field) {
	if l.level.Load() > int32(FatalLevel) {
		return
	}
	l.log(FatalLevel, msg, fields...)
	osExit(1)
}

// FormatStructAsJSON marshals v as indented JSON and writes it raw to every
// output writer, bypassing level filtering and the standard entry shape.
func (l *loggerImpl) FormatStructAsJSON(v interface{}) {

	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return
	}
	b = append(b, '\n')
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, w := range l.out {
		_, _ = w.Write(b)
	}
}

func (l *loggerImpl) log(level Level, msg string, fields ...Field) {

	if l.sampler != nil && !l.sampler.ShouldLog(level) {
		return
	}

	l.mu.Lock()

	snap := loggerSnapshot{
		layer:      l.layer,
		fields:     append([]Field(nil), l.fields...),
		jsonOutput: l.jsonOutput,
		prettyJSON: l.prettyJSON,
		out:        append([]io.Writer(nil), l.out...),
		showCaller: l.showCaller,
		hooks:      append([]Hook(nil), l.hooks...),
		ctx:        l.ctx,
		extractors: append([]Extractor(nil), l.extractors...),
	}

	var caller string
	if snap.showCaller {
		if pc, file, line, ok := runtime.Caller(2); ok {
			if fn := runtime.FuncForPC(pc); fn != nil {
				caller = fmt.Sprintf("%s:%d %s", filepath.Base(file), line, fn.Name())
			} else {
				caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
			}
		}
	}

	// Extraer campos del contexto
	extraFields := extractFields(snap.ctx, snap.extractors)
	allFields := append(append([]Field(nil), snap.fields...), extraFields...)
	allFields = append(allFields, fields...)

	// Async: encolar o fallback sincrono
	if l.async && l.queue != nil {
		select {
		case l.queue <- func() {
			l.writeLog(snap, level, msg, allFields, caller)
		}:
			l.mu.Unlock()
			return
		default:

		}
	}

	l.writeLog(snap, level, msg, allFields, caller)
	l.mu.Unlock()

}

func (l *loggerImpl) writeLog(snap loggerSnapshot, level Level, msg string, fields []Field, caller string) {
	entry := map[string]any{
		"ts":  time.Now().Format(time.RFC3339),
		"lvl": levelNames[level],
		"msg": msg,
	}

	if snap.layer != "" {
		entry["layer"] = snap.layer
	}

	if caller != "" {
		entry["caller"] = caller
	}

	for _, f := range fields {
		entry[f.Key] = f.Value
	}

	var logOutput string
	if snap.jsonOutput {
		safe := make(map[string]interface{}, len(entry))
		for k, v := range entry {
			safe[k] = v
		}
		logOutputBytes, _ := utils.FormatJSON(safe, snap.prettyJSON)
		logOutput = string(logOutputBytes)
	} else {
		logOutput = utils.FormatText(toStringKeyAny(entry), levelNames[level], snap.prettyJSON)
	}

	for _, w := range snap.out {
		_, _ = w.Write([]byte(logOutput + "\n"))
	}

	// Fire hooks
	if len(snap.hooks) > 0 {
		hookEntry := Entry{
			Time:    time.Now(),
			Level:   level,
			Message: msg,
			Layer:   snap.layer,
			Fields:  fields,
		}
		if caller != "" {
			hookEntry.Caller = caller
		}
		for _, h := range snap.hooks {
			if hookLevelsContain(h.Levels(), level) {
				_ = h.Fire(hookEntry)
			}
		}
	}
}

// Sync flushes all output writers that implement Sync() error (e.g. *os.File),
// returning the first error encountered, if any. Errors from the default
// os.Stdout writer are ignored because syncing a pipe or non-terminal returns
// EINVAL and would otherwise fail spuriously in headless environments.
func (l *loggerImpl) Sync() error {
	if l.async && l.queue != nil {
		done := make(chan struct{})
		select {
		case l.queue <- func() {
			l.syncOutputs()
			close(done)
		}:
			<-done
		case <-time.After(5 * time.Second):
			return fmt.Errorf("async logger sync timeout")
		}
		return nil
	}
	return l.syncOutputs()
}

func (l *loggerImpl) syncOutputs() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	var firstErr error
	for _, w := range l.out {
		if s, ok := w.(interface{ Sync() error }); ok {
			if err := s.Sync(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// Close closes all output writers that implement io.Closer and clears the
// output list, preventing writes after close. It returns the first error
// encountered, if any.
func (l *loggerImpl) Close() error {
	if l.async && l.queue != nil {
		close(l.done)
		l.wg.Wait()
		close(l.queue)
	}
	return l.closeOutputs()
}

func (l *loggerImpl) closeOutputs() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	var firstErr error
	for _, w := range l.out {
		if c, ok := w.(io.Closer); ok {
			if err := c.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	l.out = nil
	return firstErr
}

// toStringKeyAny converts a map[string]any into a map[string]interface{} for
// the utils package.
func toStringKeyAny(m map[string]any) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
