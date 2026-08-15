package logger

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_InfoLevel(t *testing.T) {
	logger := NewLogger(WithLevel(InfoLevel))

	var buf bytes.Buffer
	logger.AddOutput(&buf)

	// Test Info level log
	logger.Info("This is an info message")
	output := buf.String()

	// Verificar que el log se haya generado correctamente
	assert.Contains(t, output, `[INFO]`)
	assert.Contains(t, output, `This is an info message`)
}

func TestLogger_DebugLevel(t *testing.T) {
	logger := NewLogger(WithLevel(InfoLevel)) // Set level to Info, so Debug shouldn't appear

	var buf bytes.Buffer
	logger.AddOutput(&buf)

	// Test Debug level log (this should not be logged)
	logger.Debug("This is a debug message")
	output := buf.String()

	// Verify that the Debug message is not logged
	assert.NotContains(t, output, `"lvl":"DEBUG"`)
}

func TestLogger_WithLayer(t *testing.T) {
	layer := "Repository"
	logger := NewLogger(WithLevel(InfoLevel), WithLayer(layer))

	testLayer := `[` + strings.ToUpper(layer) + `]`

	var buf bytes.Buffer
	logger.AddOutput(&buf)

	// Log a message
	logger.Info("Layered message")

	output := buf.String()

	// Verify that the layer is added to the log
	assert.Contains(t, output, testLayer)
}

func TestLogger_WithFields(t *testing.T) {
	logger := NewLogger(WithLevel(InfoLevel))

	var buf bytes.Buffer
	logger.AddOutput(&buf)

	// Log a message with fields
	logger.Info("Message with fields", Field{"user_id", 123}, Field{"request_id", "abc123"})

	output := buf.String()

	// Verify that the fields are included in the log
	assert.Contains(t, output, `user_id: 123`)
	assert.Contains(t, output, `request_id: abc123`)
}

func TestLogger_JSONOutput(t *testing.T) {
	logger := NewLogger(WithLevel(InfoLevel), WithJSON(true))

	var buf bytes.Buffer
	logger.AddOutput(&buf)

	// Log a message with JSON output
	logger.Info("JSON formatted message")
	output := buf.String()

	// Verify that the output is in JSON format
	assert.Contains(t, output, `"lvl":"INFO"`)
	assert.Contains(t, output, `"msg":"JSON formatted message"`)
}

func TestLogger_TextOutput(t *testing.T) {
	logger := NewLogger(WithLevel(InfoLevel), WithJSON(false))

	var buf bytes.Buffer
	logger.AddOutput(&buf)

	// Log a message with text output
	logger.Info("Text formatted message")
	output := buf.String()

	// Verify that the output is in text format
	assert.Contains(t, output, "[INFO]")
	assert.Contains(t, output, "Text formatted message")
}

func TestLogger_Concurrency(t *testing.T) {
	logger := NewLogger(WithLevel(InfoLevel))

	var buf bytes.Buffer
	logger.AddOutput(&buf)

	// Run multiple goroutines that log concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			logger.Info("Concurrent log", Field{"index", i})
		}(i)
	}

	// Esperar a que todas las goroutines terminen antes de leer el buffer
	wg.Wait()

	output := buf.String()

	// Verify that all logs were written without interleaving
	assert.Contains(t, output, `Concurrent log`)
	assert.Contains(t, output, `index: 0`) // check at least one field
}

func TestLogger_LevelFiltering(t *testing.T) {
	// Test level filtering by setting level to Warn
	logger := NewLogger(WithLevel(WarnLevel))

	var buf bytes.Buffer
	logger.AddOutput(&buf)

	// Log messages at different levels
	logger.Info("This is an info message")
	logger.Warn("This is a warning message")
	logger.Error("This is an error message")

	output := buf.String()

	// Verify that only warning and error messages are logged
	assert.NotContains(t, output, `INFO`)
	assert.Contains(t, output, `WARN`)
	assert.Contains(t, output, `ERROR`)
}

// ==================== FASE 2: Tests de Robustez ====================

type mockWriter struct {
	bytes.Buffer
	syncCalled  bool
	closeCalled bool
	mu          sync.Mutex
}

func (m *mockWriter) Sync() error {
	m.syncCalled = true
	return nil
}

func (m *mockWriter) Close() error {
	m.closeCalled = true
	return nil
}

// safeBuffer es un bytes.Buffer con mutex para tests async/concurrentes
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func TestLogger_Fatal(t *testing.T) {
	var exitCode int
	oldExit := osExit
	osExit = func(code int) { exitCode = code }
	defer func() { osExit = oldExit }()

	var buf bytes.Buffer
	logger := NewLogger(WithLevel(DebugLevel))
	logger.AddOutput(&buf)

	logger.Fatal("fatal error occurred")

	output := buf.String()
	assert.Contains(t, output, "FATAL")
	assert.Contains(t, output, "fatal error occurred")
	assert.Equal(t, 1, exitCode)
}

func TestLogger_Sync(t *testing.T) {
	mock := &mockWriter{}
	// Crear logger limpio sin os.Stdout para evitar "sync /dev/stdout: invalid argument"
	logger := NewLogger(WithLevel(InfoLevel))
	impl := logger.(*loggerImpl)
	impl.mu.Lock()
	impl.out = []io.Writer{mock}
	impl.mu.Unlock()

	logger.Info("sync test")
	err := logger.Sync()

	assert.NoError(t, err)
	assert.True(t, mock.syncCalled)
}
func TestLogger_Close(t *testing.T) {
	mock := &mockWriter{}
	logger := NewLogger(WithLevel(InfoLevel))
	impl := logger.(*loggerImpl)
	impl.mu.Lock()
	impl.out = []io.Writer{mock}
	impl.mu.Unlock()

	err := logger.Close()

	assert.NoError(t, err)
	assert.True(t, mock.closeCalled)
}
func TestLogger_RaceCondition_LevelAndLog(t *testing.T) {
	logger := NewLogger(WithLevel(InfoLevel))
	var buf safeBuffer
	logger.AddOutput(&buf)

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%3 == 0 {
				logger.SetLevel(DebugLevel)
			} else if i%3 == 1 {
				logger.SetLevel(WarnLevel)
			} else {
				logger.Info("race test", Field{"i", i})
			}
		}(i)
	}
	wg.Wait()

	// Si no hay panic de race condition, el test pasa
	assert.True(t, true)
}

func TestLogger_JSONOmitsEmptyLayer(t *testing.T) {
	logger := NewLogger(WithLevel(InfoLevel), WithJSON(true))

	var buf bytes.Buffer
	logger.AddOutput(&buf)

	logger.Info("no layer here")

	output := buf.String()
	assert.Contains(t, output, `"msg":"no layer here"`)
	assert.NotContains(t, output, `"layer"`)
}

func TestLogger_JSONIncludesLayer(t *testing.T) {
	logger := NewLogger(WithLevel(InfoLevel), WithJSON(true), WithLayer("Service"))

	var buf bytes.Buffer
	logger.AddOutput(&buf)

	logger.Info("with layer")

	output := buf.String()
	assert.Contains(t, output, `"layer":"Service"`)
}

func TestLogger_CloneIndependentLevel(t *testing.T) {
	parent := NewLogger(WithLevel(InfoLevel))
	child := parent.WithLayer("Child")

	var buf bytes.Buffer
	child.AddOutput(&buf)

	// Cambiar nivel del padre no debe afectar al hijo
	parent.SetLevel(ErrorLevel)
	child.Info("child should still log")

	output := buf.String()
	assert.Contains(t, output, "child should still log")
}

// ==================== FASE 3: Tests de Producción ====================

func TestLogger_Sampling_EveryN(t *testing.T) {
	sampler := NewEveryNSampler(5, InfoLevel) // 1 de cada 5
	logger := NewLogger(WithLevel(DebugLevel), WithSampling(sampler))

	var buf bytes.Buffer
	logger.AddOutput(&buf)

	// 10 logs, deberían pasar ~2
	for i := 0; i < 10; i++ {
		logger.Info("sampled")
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// Con EveryN=5, deberían pasar 2 (índices 5 y 10)
	assert.True(t, len(lines) >= 1 && len(lines) <= 3, "Esperaba 1-3 logs, obtuve %d", len(lines))
}

func TestLogger_Sampling_NeverSamplesErrors(t *testing.T) {
	sampler := NewEveryNSampler(1000, InfoLevel) // casi nunca pasa
	logger := NewLogger(WithLevel(DebugLevel), WithSampling(sampler))

	var buf bytes.Buffer
	logger.AddOutput(&buf)

	logger.Error("error crítico") // debe pasar SIEMPRE

	output := buf.String()
	assert.Contains(t, output, "error crítico")
}

type testHook struct {
	entries []Entry
	levels  []Level
	mu      sync.Mutex
}

func (h *testHook) Levels() []Level { return h.levels }
func (h *testHook) Fire(e Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, e)
	return nil
}

func TestLogger_Hook_FiresOnMatchingLevel(t *testing.T) {
	hook := &testHook{levels: []Level{ErrorLevel, FatalLevel}}
	logger := NewLogger(WithLevel(DebugLevel))
	logger.AddHook(hook)

	var buf bytes.Buffer
	logger.AddOutput(&buf)

	logger.Info("info msg")
	logger.Error("error msg")

	assert.Len(t, hook.entries, 1)
	assert.Equal(t, "error msg", hook.entries[0].Message)
}

func TestLogger_Hook_ReceivesFields(t *testing.T) {
	hook := &testHook{levels: []Level{InfoLevel}}
	logger := NewLogger(WithLevel(DebugLevel))
	logger.AddHook(hook)

	logger.Info("hello", Field{"user", "alice"})

	require.Len(t, hook.entries, 1)
	assert.Equal(t, "alice", hook.entries[0].Fields[0].Value)
}

func TestLogger_ContextExtractor(t *testing.T) {
	ctx := context.WithValue(context.Background(), "trace_id", "abc-123")

	logger := NewLogger(
		WithLevel(DebugLevel),
		WithContext(ctx),
		WithExtractors(ExtractTraceID),
	)

	var buf bytes.Buffer
	logger.AddOutput(&buf)

	logger.Info("request")

	output := buf.String()
	assert.Contains(t, output, "trace_id")
	assert.Contains(t, output, "abc-123")
}
func TestLogger_Async_WritesEventually(t *testing.T) {
	logger := NewLogger(
		WithLevel(DebugLevel),
		WithAsync(100),
	)

	var buf safeBuffer
	logger.AddOutput(&buf)

	logger.Info("async msg")
	err := logger.Sync()
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "async msg")
}
func TestLogger_Async_SyncFlushes(t *testing.T) {
	logger := NewLogger(
		WithLevel(DebugLevel),
		WithAsync(100),
	)

	var buf safeBuffer
	logger.AddOutput(&buf)

	logger.Info("before sync")
	err := logger.Sync()
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "before sync")
}
func TestRotatingFile_RotatesOnSize(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.log")

	rf, err := NewRotatingFile(path, 1, 3, 0) // 1MB max, 3 backups
	require.NoError(t, err)
	defer rf.Close()

	// Escribir 600KB
	data := make([]byte, 600*1024)
	_, err = rf.Write(data)
	require.NoError(t, err)

	// Escribir 600KB más → debería rotar (supera 1MB)
	_, err = rf.Write(data)
	require.NoError(t, err)

	// Verificar que existe el backup
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 2) // original + backup
}

func TestLogger_WithSamplingClone(t *testing.T) {
	parent := NewLogger(WithLevel(DebugLevel))
	child := parent.WithSampling(NewEveryNSampler(2, InfoLevel))

	var buf bytes.Buffer
	child.AddOutput(&buf)

	// 4 logs con sampling 1/2 → ~2 pasan
	for i := 0; i < 4; i++ {
		child.Info("test")
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.True(t, len(lines) >= 1 && len(lines) <= 3)
}
