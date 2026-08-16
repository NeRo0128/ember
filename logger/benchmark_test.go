package logger

import (
	"io"
	"testing"
)

func BenchmarkLogger_Info(b *testing.B) {
	log := NewLogger(WithLevel(InfoLevel))
	log.AddOutput(io.Discard)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		log.Info("benchmark message", Field{Key: "i", Value: i})
	}
}

func BenchmarkLogger_InfoJSON(b *testing.B) {
	log := NewLogger(WithLevel(InfoLevel), WithJSON(true))
	log.AddOutput(io.Discard)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		log.Info("benchmark message", Field{Key: "i", Value: i})
	}
}

func BenchmarkLogger_InfoAsync(b *testing.B) {
	log := NewLogger(WithLevel(InfoLevel), WithAsync(1000))
	log.AddOutput(io.Discard)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		log.Info("benchmark message", Field{Key: "i", Value: i})
	}
	log.Close()
}
