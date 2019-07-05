/*
Package http is used to generate common HTTP responses.

Use StatusCodeAndResponse(...) in HTTP handlers to return a common JSON response.
	{
		error: {
			"code" : "<code passed to ctxerr.New/Wrap>",
			"action" : "<value under the field key ctxerr.FieldKeyAction>",
			"messsage" : "error.Error()",
			"traceID" : "<opencensus trace ID>",
			"fields" : {},
		}
	}


If you are using net/http:
	if err != nil {
		statusCode, response := StatusCodeAndResponse(err, config.ShowErrorMessage, config.ShowErrorFields)
		w.WriteHeader(statusCode)
		b, err := json.Marshal(response)
		if err != nil {
			ctxerr.LogError(err)
		}
		w.Write(response)
		return
	}

If you are using echo:
	if err != nil {
		statusCode, response := StatusCodeAndResponse(err, config.ShowErrorMessage, config.ShowErrorFields)
		return c.JSON(statusCode, response)
	}
*/
package http

import (
	"context"
	"strconv"

	"go.opencensus.io/trace"
	"github.com/mvndaai/ctxerr"
)

type (
	// ErrorResponse is the default HTTP response
	ErrorResponse struct {
		Error Details `json:"error"`
	}

	// Details of a response
	Details struct {
		TraceID string                 `json:"traceID,omitempty"`
		Code    string                 `json:"code"`
		Action  string                 `json:"action,omitempty"`
		Message string                 `json:"messsage,omitempty"`
		Fields  map[string]interface{} `json:"fields,omitempty"`
	}
)

// StatusCodeAndResponse extracts info from the error to create a standard response
func StatusCodeAndResponse(err error, showMessage, showFields bool) (int, ErrorResponse) {
	statusCode := 500
	r := ErrorResponse{}

	if showMessage {
		if err != nil {
			r.Error.Message = err.Error()
		}
	}

	if de := ctxerr.Deepest(err); de != nil {
		r.Error.TraceID = TraceID(de.Context())

		fields := de.Fields()
		if showFields {
			r.Error.Fields = fields
		}
		if code, ok := fields[ctxerr.FieldKeyCode]; ok {
			r.Error.Code = code.(string)
		}
		if action, ok := fields[ctxerr.FieldKeyAction]; ok {
			r.Error.Action = action.(string)
		}
		if sci, ok := fields[ctxerr.FieldKeyStatusCode]; ok {
			switch v := sci.(type) {
			case int:
				statusCode = v
			case string:
				sc, err := strconv.Atoi(v)
				if err != nil {
					ctx := ctxerr.SetField(de.Context(), ctxerr.FieldKeyRelatedCode, fields[ctxerr.FieldKeyCode])
					err = ctxerr.Wrap(ctx, err, "d81e453f-ce91-43a4-a443-404873b94c15",
						"could not convert s")
					ctxerr.LogWarn(err)
					break
				}
				statusCode = sc
			default:
				ctx := ctxerr.SetField(de.Context(), ctxerr.FieldKeyRelatedCode, fields[ctxerr.FieldKeyCode])
				err := ctxerr.New(ctx, "594b85d2-0558-4335-9792-1117c22329be",
					"could not convert field '%s:%v' type '%T' to int", ctxerr.FieldKeyStatusCode, v, v)
				ctxerr.LogWarn(err)
			}
		}
	}

	return statusCode, r
}

// TraceID uses opencensus to get the trace ID from the context
func TraceID(ctx context.Context) string {
	if span := trace.FromContext(ctx); span != nil {
		return span.SpanContext().TraceID.String()
	}
	return ""
}
