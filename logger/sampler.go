package logger

import "sync/atomic"

// Sampler decides whether a log entry should be kept or discarded for
// sampling purposes.
type Sampler interface {
	// ShouldLog reports whether the entry at the given level should be kept.
	ShouldLog(level Level) bool
}

// EveryNSampler keeps 1 out of every N log entries, but only for levels at or
// below MaxLevel. It never samples levels above MaxLevel, so errors always
// pass through.
type EveryNSampler struct {
	// EveryN is the sampling period: keep every EveryN-th entry.
	EveryN int64
	// MaxLevel is the highest level subject to sampling; levels above it are
	// always kept.
	MaxLevel Level // sampling only applies to levels <= MaxLevel
	counter  atomic.Int64
}

// NewEveryNSampler creates an EveryNSampler. everyN must be >= 1 (values below
// 1 are clamped to 1). maxLevel limits sampling to that level and below (e.g.
// InfoLevel), so higher severities are never dropped.
func NewEveryNSampler(everyN int, maxLevel Level) *EveryNSampler {
	if everyN <= 0 {
		everyN = 1
	}
	return &EveryNSampler{
		EveryN:   int64(everyN),
		MaxLevel: maxLevel,
	}
}

// ShouldLog reports whether the entry should be kept. Entries above MaxLevel
// are always kept; the rest are kept every EveryN calls using an atomic
// counter, making it safe for concurrent use.
func (s *EveryNSampler) ShouldLog(level Level) bool {
	if level > s.MaxLevel {
		return true
	}
	return s.counter.Add(1)%s.EveryN == 0
}
