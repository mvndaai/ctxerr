/*
Package opencensus can be imported to use opencensus for tracing

As a side effect of importing the package the http.TraceID function gets replaced

import _ "github.com/mvndaai/ctxerr/http/trace/opencensus"

*/
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
