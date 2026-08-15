package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/NeRo0128/logcore/internal/utils"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

var levelNames = map[Level]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
	FatalLevel: "FATAL",
}

// Logger Interface
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)

	FormatStructAsJSON(v interface{})

	WithFields(fields ...Field) Logger
	WithLayer(layer string) Logger
	WithContext(ctx context.Context) Logger

	SetLevel(level Level)
	AddOutput(w io.Writer)
}

// Field representa un campo en el log
type Field struct {
	Key   string
	Value any
}

type loggerImpl struct {
	mu         sync.Mutex
	level      Level
	layer      string
	fields     []Field
	jsonOutput bool
	prettyJSON bool
	out        []io.Writer
	ctx        context.Context
	showCaller bool
}

func NewLogger(options ...Option) Logger {
	l := &loggerImpl{
		level:      InfoLevel,
		out:        []io.Writer{os.Stdout},
		jsonOutput: false,
		prettyJSON: false,
	}
	for _, option := range options {
		option(l)
	}
	return l
}

func NewDebugLogger(layer string) Logger {
	l := &loggerImpl{
		level:      DebugLevel,
		out:        []io.Writer{os.Stdout},
		jsonOutput: false,
		prettyJSON: false,
		layer:      layer,
	}
	return l
}

// Logger Methods

func (l *loggerImpl) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *loggerImpl) AddOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = append(l.out, w)
}

func (l *loggerImpl) WithFields(fields ...Field) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	clone := &loggerImpl{
		level:      l.level,
		layer:      l.layer,
		fields:     append(append([]Field(nil), l.fields...), fields...),
		jsonOutput: l.jsonOutput,
		prettyJSON: l.prettyJSON,
		out:        append([]io.Writer(nil), l.out...),
		ctx:        l.ctx,
	}
	return clone
}

func (l *loggerImpl) WithContext(ctx context.Context) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	clone := &loggerImpl{
		level:      l.level,
		layer:      l.layer,
		fields:     append([]Field(nil), l.fields...),
		jsonOutput: l.jsonOutput,
		prettyJSON: l.prettyJSON,
		out:        append([]io.Writer(nil), l.out...),
		ctx:        ctx,
	}
	return clone
}

func (l *loggerImpl) WithLayer(layer string) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	clone := &loggerImpl{
		level:      l.level,
		layer:      layer,
		fields:     append([]Field(nil), l.fields...),
		jsonOutput: l.jsonOutput,
		prettyJSON: l.prettyJSON,
		out:        append([]io.Writer(nil), l.out...),
		ctx:        l.ctx,
	}
	return clone
}

// Writers

func (l *loggerImpl) Debug(msg string, fields ...Field) {
	if l.level > DebugLevel {
		return
	}
	l.log(DebugLevel, msg, fields...)
}

func (l *loggerImpl) Info(msg string, fields ...Field) {
	if l.level > InfoLevel {
		return
	}
	l.log(InfoLevel, msg, fields...)
}

func (l *loggerImpl) Warn(msg string, fields ...Field) {
	if l.level > WarnLevel {
		return
	}
	l.log(WarnLevel, msg, fields...)
}

func (l *loggerImpl) Error(msg string, fields ...Field) {
	if l.level > ErrorLevel {
		return
	}
	l.log(ErrorLevel, msg, fields...)
}

func (l *loggerImpl) Fatal(msg string, fields ...Field) {
	if l.level > FatalLevel {
		return
	}
	l.log(FatalLevel, msg, fields...)
}

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
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := map[string]any{
		"ts":    time.Now().Format(time.RFC3339),
		"lvl":   levelNames[level],
		"msg":   msg,
		"layer": l.layer,
	}

	// Añadir caller si está habilitado
	if l.showCaller {
		// skip 3 frames: runtime.Callers -> this function -> caller
		if pc, file, line, ok := runtime.Caller(2); ok {
			fn := runtime.FuncForPC(pc)
			if fn != nil {
				entry["caller"] = fmt.Sprintf("%s:%d %s", filepath.Base(file), line, fn.Name())
			} else {
				entry["caller"] = fmt.Sprintf("%s:%d", filepath.Base(file), line)
			}
		}
	}

	for _, f := range append(l.fields, fields...) {
		entry[f.Key] = f.Value
	}

	var logOutput string
	if l.jsonOutput {
		safe := make(map[string]interface{}, len(entry))
		for k, v := range entry {
			safe[k] = v
		}
		logOutputBytes, _ := utils.FormatJSON(safe, l.prettyJSON)
		logOutput = string(logOutputBytes)
	} else {
		logOutput = utils.FormatText(toStringKeyAny(entry), levelNames[level], l.prettyJSON)

	}

	for _, w := range l.out {
		_, _ = w.Write([]byte(logOutput + "\n"))
	}
}

// helper: convertir map[string]any -> map[string]interface{} para utils
func toStringKeyAny(m map[string]any) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
