package http_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/mvndaai/ctxerr"
	"github.com/mvndaai/ctxerr/http"
)

func TestStatusCodeAndResponse(t *testing.T) {
	defaultStatusCode := 500
	happyCode := "code"
	happyMessage := "message"

	tests := []struct {
		name        string
		showMessage bool
		showFields  bool
		err         error

		expectedStatusCode int
		expectedCode       string
		expectedAction     string
		expectedMessage    string
		expectedFields     map[string]interface{}
		expectedWarnings   bool
	}{
		{
			name:        "go",
			showMessage: true,
			showFields:  true,
			err:         errors.New(happyMessage),

			expectedCode:       "",
			expectedStatusCode: defaultStatusCode,
			expectedAction:     "",
			expectedMessage:    happyMessage,
			expectedFields:     nil,
		},
		{
			name:        "hide message",
			showMessage: false,
			showFields:  true,
			err:         errors.New(happyMessage),

			expectedStatusCode: defaultStatusCode,
			expectedCode:       "",
			expectedAction:     "",
			expectedMessage:    "",
			expectedFields:     nil,
		},
		{
			name:        "ctxerr",
			showMessage: true,
			showFields:  true,
			err:         ctxerr.New(context.Background(), happyCode, happyMessage),

			expectedStatusCode: defaultStatusCode,
			expectedCode:       happyCode,
			expectedAction:     "",
			expectedMessage:    happyMessage,
			expectedFields:     nil,
		},
		{
			name:        "action",
			showMessage: true,
			showFields:  true,
			err: func() error {
				ctx := ctxerr.SetField(context.Background(), ctxerr.FieldKeyAction, "action")
				return ctxerr.New(ctx, happyCode, happyMessage)
			}(),
			expectedStatusCode: defaultStatusCode,
			expectedCode:       happyCode,
			expectedAction:     "action",
			expectedMessage:    happyMessage,
			expectedFields:     nil,
		},
		{
			name: "status code int",
			err: func() error {
				ctx := ctxerr.SetField(context.Background(), ctxerr.FieldKeyStatusCode, 400)
				return ctxerr.New(ctx, happyCode, happyMessage)
			}(),

			expectedStatusCode: 400,
			expectedCode:       happyCode,
		},
		{
			name: "status code int string",
			err: func() error {
				ctx := ctxerr.SetField(context.Background(), ctxerr.FieldKeyStatusCode, "400")
				return ctxerr.New(ctx, happyCode, happyMessage)
			}(),

			expectedStatusCode: 400,
			expectedCode:       happyCode,
		},
		{
			name: "status code other string",
			err: func() error {
				ctx := ctxerr.SetField(context.Background(), ctxerr.FieldKeyStatusCode, "foo")
				return ctxerr.New(ctx, happyCode, happyMessage)
			}(),

			expectedStatusCode: 500,
			expectedCode:       happyCode,
		},
		{
			name: "status code typed int",
			err: func() error {
				ctx := ctxerr.SetField(context.Background(), ctxerr.FieldKeyStatusCode, int64(400))
				return ctxerr.New(ctx, happyCode, happyMessage)
			}(),

			expectedStatusCode: 400,
			expectedCode:       happyCode,
		},
		{
			name: "status code other",
			err: func() error {
				ctx := ctxerr.SetField(context.Background(), ctxerr.FieldKeyStatusCode, true)
				return ctxerr.New(ctx, happyCode, happyMessage)
			}(),

			expectedStatusCode: 500,
			expectedCode:       happyCode,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sc, r := http.StatusCodeAndResponse(test.err, test.showMessage, test.showFields)

			if sc != test.expectedStatusCode {
				t.Error("Status code did not match", sc, test.expectedStatusCode)
			}
			if v := r.Error.Code; v != test.expectedCode {
				t.Error("Code did not match", v, test.expectedCode)
			}
			if v := r.Error.Action; v != test.expectedAction {
				t.Error("Action did not match", v, test.expectedAction)
			}
			if v := r.Error.Message; v != test.expectedMessage {
				t.Error("Message did not match", v, test.expectedMessage)
			}

			// Ignore location field in test
			delete(r.Error.Fields, ctxerr.FieldKeyLocation)

			fs := fmt.Sprint(test.expectedFields)
			if v := fmt.Sprint(r.Error.Fields); v != fs {
				t.Error("Fields did not match", v, fs)
			}
		})
	}
}
