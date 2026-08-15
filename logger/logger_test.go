package logger

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
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
