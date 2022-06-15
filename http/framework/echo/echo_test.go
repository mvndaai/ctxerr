package echo_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	"github.com/mvndaai/ctxerr"
	ctxhttp "github.com/mvndaai/ctxerr/http"
	ctxecho "github.com/mvndaai/ctxerr/http/framework/echo"
)

func TestErrorHandler(t *testing.T) {
	code := "code"
	message := "message"

	tests := []struct {
		name            string
		toErr           func(context.Context) error
		expectedCode    string
		expectedMessage string
	}{
		{
			name: "ctxerr",
			toErr: func(ctx context.Context) error {
				return ctxerr.New(ctx, code, message)
			},
			expectedCode:    code,
			expectedMessage: message,
		},
		{
			name: "go error",
			toErr: func(ctx context.Context) error {
				return errors.New(message)
			},
			expectedCode:    "",
			expectedMessage: message,
		},
		{
			name: "echo error",
			toErr: func(ctx context.Context) error {
				return &echo.HTTPError{Message: message}
			},
			expectedCode:    "",
			expectedMessage: message,
		},
	}

	e := echo.New()
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handled := false
	ctxerr.AddHandleHook(func(_ error) { handled = true })

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handled = false

			eh := ctxecho.ErrorHandler(true, false)
			handler := func(c echo.Context) error {
				return test.toErr(c.Request().Context())
			}
			eh(handler(c), c)

			if !handled {
				t.Error("Error not handled")
			}

			var response ctxhttp.ErrorResponse
			b, err := ioutil.ReadAll(rec.Body)
			if err != nil {
				t.Error("Could not read recorded body", err)
			}
			t.Log(string(b))

			if err := json.Unmarshal(b, &response); err != nil {
				t.Error("respons did not marshall into JSON", err)
			}
			if v := response.Error.Code; v != test.expectedCode {
				t.Error("Code in response did not match", v, code)
			}
		})
	}
}
