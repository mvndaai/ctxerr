package opencensus

import (
	"context"

	"github.com/mvndaai/ctxerr/http"
	"go.opencensus.io/trace"
)

func init() {
	http.TraceID = TraceID
}

// TraceID uses opencensus to get the trace ID from the context
func TraceID(ctx context.Context) string {
	if span := trace.FromContext(ctx); span != nil {
		return span.SpanContext().TraceID.String()
	}
	return ""
}
