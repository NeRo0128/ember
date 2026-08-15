package logger

import "context"

// Option is a functional option that configures a Logger at construction time.
type Option func(*loggerImpl)

// WithJSON enables or disables JSON output for the logger.
func WithJSON(enabled bool) Option {
	return func(l *loggerImpl) {
		l.jsonOutput = enabled
	}
}

// WithPrettyJSON enables or disables indented (pretty) JSON output. It has no
// effect unless JSON output is enabled.
func WithPrettyJSON(enabled bool) Option {
	return func(l *loggerImpl) {
		l.prettyJSON = enabled
	}
}

// WithLevel sets the minimum severity that the logger will emit.
func WithLevel(level Level) Option {
	return func(l *loggerImpl) {
		l.level.Store(int32(level))
	}
}

// WithLayer sets the layer name attached to every log entry.
func WithLayer(layer string) Option {
	return func(l *loggerImpl) {
		l.layer = layer
	}
}

// WithField attaches a single field to every log entry of the logger.
func WithField(field Field) Option {
	return func(l *loggerImpl) {
		l.fields = append(l.fields, field)
	}
}

// WithCaller enables or disables annotating entries with the caller's file,
// line, and function name.
func WithCaller(enable bool) Option {
	return func(l *loggerImpl) {
		l.showCaller = enable
	}
}

// WithSampling attaches a Sampler that decides which log entries are kept,
// discarding the rest.
func WithSampling(sampler Sampler) Option {
	return func(l *loggerImpl) {
		l.sampler = sampler
	}
}

func WithAsync(queueSize int) Option {
	return func(l *loggerImpl) {
		if queueSize <= 0 {
			queueSize = 1000
		}
		l.async = true
		l.queue = make(chan func(), queueSize)
		l.done = make(chan struct{})
		l.wg.Add(1)
		go l.worker()
	}
}

func WithContext(ctx context.Context) Option {
	return func(l *loggerImpl) {
		l.ctx = ctx
	}
}

// WithExtractors añade extractores de contexto al logger.
func WithExtractors(extractors ...Extractor) Option {
	return func(l *loggerImpl) {
		l.extractors = append(l.extractors, extractors...)
	}
}