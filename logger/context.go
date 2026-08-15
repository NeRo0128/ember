package logger

import "context"

// Extractor pulls a Field from a context.Context, returning nil when the key
// is missing or has an unexpected type.
type Extractor func(ctx context.Context) *Field

// extractFields runs every extractor against a context and collects the
// resulting fields.
func extractFields(ctx context.Context, extractors []Extractor) []Field {
	if ctx == nil || len(extractors) == 0 {
		return nil
	}
	var fields []Field
	for _, ext := range extractors {
		if f := ext(ctx); f != nil {
			fields = append(fields, *f)
		}
	}
	return fields
}

// ExtractTraceID builds a "trace_id" field from the value stored under the
// "trace_id" context key, or nil when absent or not a non-empty string.
func ExtractTraceID(ctx context.Context) *Field {
	if id, ok := ctx.Value("trace_id").(string); ok && id != "" {
		return &Field{Key: "trace_id", Value: id}
	}
	return nil
}

// ExtractSpanID builds a "span_id" field from the value stored under the
// "span_id" context key, or nil when absent or not a non-empty string.
func ExtractSpanID(ctx context.Context) *Field {
	if id, ok := ctx.Value("span_id").(string); ok && id != "" {
		return &Field{Key: "span_id", Value: id}
	}
	return nil
}

// ExtractUserID builds a "user_id" field from the value stored under the
// "user_id" context key, or nil when absent or not a non-empty string.
func ExtractUserID(ctx context.Context) *Field {
	if id, ok := ctx.Value("user_id").(string); ok && id != "" {
		return &Field{Key: "user_id", Value: id}
	}
	return nil
}
