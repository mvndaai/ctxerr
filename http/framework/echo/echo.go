/*
Package echo has functions to use with echo (https://echo.labstack.com).
	import ctxecho "github.com/mvndaai/ctxerr/http/framework/echo"

	func main() {
		...
		e.HTTPErrorHandler = ctxecho.ErrorHandler(config.ShowMessage, config.ShowFields)
		...
	}
*/
package echo

import (
	"fmt"

	"github.com/labstack/echo"
	"github.com/mvndaai/ctxerr"
	"github.com/mvndaai/ctxerr/http"
)

// ErrorHandler implements an echo Custom  HTTP Error Handler.
// This uses the ctxerr/http package to return a standardized response.
// See https://echo.labstack.com/guide/error-handling for more information on error handlers.
func ErrorHandler(showMessage, showFields bool) func(err error, c echo.Context) {

	return func(err error, c echo.Context) {
		ctxerr.Handle(err)
		statusCode, response := http.StatusCodeAndResponse(err, showMessage, showFields)

		// Catch 404s or other routing errors
		if he, ok := err.(*echo.HTTPError); ok {
			statusCode = he.Code
			if showMessage {
				response.Error.Message = fmt.Sprintf("%s", he.Message)
			}
		}

		if response.Error.TraceID == "" {
			response.Error.TraceID = http.TraceID(c.Request().Context())
		}

		c.JSON(statusCode, response)
	}
}
