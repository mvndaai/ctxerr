/*
Package http is used to generate common HTTP responses.

Use StatusCodeAndResponse(...) in HTTP handlers to return a common JSON response.
	{
		"error": {
			"code" : "<code passed to ctxerr.New/Wrap>",
			"action" : "<value under the field key ctxerr.FieldKeyAction>",
			"messsage" : "error.Error()",
			"traceID" : "<trace ID, if configured>",
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
	"fmt"
	"strconv"

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

	if ce, ok := ctxerr.As(err); ok {
		r.Error.TraceID = TraceID(ce.Context())
	}

	fields := ctxerr.AllFields(err)
	if len(fields) > 0 {
		if code, ok := fields[ctxerr.FieldKeyCode]; ok {
			r.Error.Code = code.(string)
			delete(fields, ctxerr.FieldKeyCode)
		}
		if action, ok := fields[ctxerr.FieldKeyAction]; ok {
			r.Error.Action = action.(string)
			delete(fields, ctxerr.FieldKeyAction)
		}
		if sci, ok := fields[ctxerr.FieldKeyStatusCode]; ok {
			switch v := sci.(type) {
			case int:
				statusCode = v
				delete(fields, ctxerr.FieldKeyStatusCode)
			default:
				sc, err := strconv.Atoi(fmt.Sprint(v))
				if err != nil {
					ctx := ctxerr.SetField(context.Background(), "related_error_code", fields[ctxerr.FieldKeyCode])
					ctx = ctxerr.SetField(ctx, "status code", v)
					ctx = ctxerr.SetField(ctx, ctxerr.FieldKeyStatusCode, 418)
					err = ctxerr.Wrap(ctx, err, "ctxerr_http", "could not convert status code to int")
					ctxerr.Handle(err)
					break
				}
				statusCode = sc
				delete(fields, ctxerr.FieldKeyStatusCode)
			}
		}
		if showFields {
			r.Error.Fields = fields
		}
	}

	return statusCode, r
}

// TraceID is a configurable function to get a traceID from the context.
var TraceID = func(ctx context.Context) string { return "" }
