package observability

import "context"

type ctxKey int

const requestIDKey ctxKey = iota

// ContextWithRequestID returns a child context that carries the request ID for logging and tracing.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext returns the request ID if present; otherwise an empty string.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}
