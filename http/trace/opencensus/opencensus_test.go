package opencensus_test

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/mvndaai/ctxerr/http"
	"go.opencensus.io/trace"
)

func TestTraceID(t *testing.T) {
	traceID := "12e3249570b71c725235bbec6d4018fa"

	tHex, err := hex.DecodeString(traceID)
	if err != nil {
		t.Fatalf("could not convert traceID (%s) to hex", traceID)
	}
	var tid trace.TraceID
	copy(tid[:], tHex)

	ctx := context.Background()
	parent := trace.SpanContext{TraceID: tid}
	ctx, _ = trace.StartSpanWithRemoteParent(ctx, "", parent)

	out := http.TraceID(ctx)
	if out != traceID {
		t.Error("Trace ID did not match", out, traceID)
	}
}
