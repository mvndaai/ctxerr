package ctxerr_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/mvndaai/ctxerr"
)

func TestFields(t *testing.T) {
	fk := "final key"
	fv := "final value"

	ctx := context.Background()
	ctx = ctxerr.SetField(ctx, "foo", "bar")
	ctx = ctxerr.SetField(ctx, fk, "baz")
	ctx = ctxerr.SetField(ctx, fk, fv)

	e := ctxerr.Deepest(ctxerr.New(ctx, "1c678a4a-305f-4f68-880f-f459009e42ee", "msg"))
	for k, v := range e.Fields() {
		if k == fk {
			if v != fv {
				t.Error("field value was incorrect: ", v)
			}
		}
	}

	if t.Failed() {
		t.Logf("fields %+v", e.Fields())
	}
}

func TestNil(t *testing.T) {
	err := ctxerr.Wrap(context.Background(), nil, "", "")
	if err != nil {
		t.Error("error should have been nil")
	}
}

func TestDeepestErr(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		isErr bool
	}{
		{},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			//TODO
		})
	}
}

func TestIntegrations(t *testing.T) {
	code := "code"
	tests := []struct {
		name  string
		toErr func(context.Context) error

		expectedMessage string
		expectedFields  map[string]interface{}
		expectedWarn    bool
	}{
		{
			name:            "nil",
			toErr:           func(ctx context.Context) error { return nil },
			expectedMessage: "<nil>",
			expectedFields:  nil,
		},
		{
			name:            "errors",
			toErr:           func(ctx context.Context) error { return errors.New("errors") },
			expectedMessage: "errors",
			expectedFields:  map[string]interface{}{},
		},
		{
			name: "ctxerr",
			toErr: func(ctx context.Context) error {
				return ctxerr.New(ctx, code, "ctxerr")
			},
			expectedMessage: "ctxerr",
			expectedFields:  map[string]interface{}{ctxerr.FieldKeyCode: code},
		},
		{
			name: "action",
			toErr: func(ctx context.Context) error {
				ctx = ctxerr.SetField(ctx, ctxerr.FieldKeyAction, "action")
				return ctxerr.New(ctx, code, "action")
			},
			expectedMessage: "action",
			expectedFields: map[string]interface{}{
				ctxerr.FieldKeyCode:   code,
				ctxerr.FieldKeyAction: "action",
			},
		},
		{
			name: "wrap",
			toErr: func(ctx context.Context) error {
				return ctxerr.Wrap(ctx, errors.New("wrapped"), code, "wrap")
			},
			expectedMessage: "wrap : wrapped",
			expectedFields:  map[string]interface{}{ctxerr.FieldKeyCode: code},
		},
		{
			name: "nil wrap",
			toErr: func(ctx context.Context) error {
				return ctxerr.Wrap(ctx, nil, code, "nil wrap")
			},
			expectedMessage: "<nil>",
			expectedFields:  map[string]interface{}{},
		},
		{
			name: "with fields",
			toErr: func(ctx context.Context) error {
				ctx = ctxerr.SetFields(ctx, map[string]interface{}{"foo": "bar"})
				return ctxerr.New(ctx, code, "with fields")
			},
			expectedMessage: "with fields",
			expectedFields:  map[string]interface{}{ctxerr.FieldKeyCode: code, "foo": "bar"},
		},
		{
			name: "new no code",
			toErr: func(ctx context.Context) error {
				return ctxerr.New(ctx, "", "new no code")
			},
			expectedMessage: "new no code",
			expectedFields:  map[string]interface{}{ctxerr.FieldKeyCode: "no_code"},
			expectedWarn:    true,
		},
	}

	var logMessage string
	var logFields string
	ctxerr.LogError = func(err error) {
		logMessage = fmt.Sprint(err)

		if ce, ok := err.(ctxerr.CtxErr); ok {
			logFields = fmt.Sprint(ce.Fields())
		}
	}

	var warning bool
	ctxerr.LogWarn = func(err error) { warning = true }

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			warning = false
			logMessage = "<nil>"
			logFields = fmt.Sprint(map[string]interface{}{})

			ctxerr.LogError(test.toErr(context.Background()))

			if logMessage != test.expectedMessage {
				t.Errorf("Message did not match log message:\n%s\n%s", logMessage, test.expectedMessage)
			}
			if tef := fmt.Sprint(test.expectedFields); logFields != tef {
				t.Errorf("Fields did not match:\n%s\n%s", logFields, tef)
			}
			if warning != test.expectedWarn {
				t.Error("Warning did not match")
			}
		})

	}
}

func TestQuickWrap(t *testing.T) {
	tests := []struct {
		name string
		err  func(context.Context) error

		expectedWarnings int
		expectedMessage  string
		expectedCode     string
	}{
		{
			name:             "external",
			err:              func(ctx context.Context) error { return ctxerr.QuickWrap(ctx, errors.New("external")) },
			expectedMessage:  "ctxerr_test.TestQuickWrap.func1 : external",
			expectedWarnings: 1,
			expectedCode:     "no_code",
		},
		{
			name: "ctxerr",
			err: func(ctx context.Context) error {
				return ctxerr.QuickWrap(ctx, ctxerr.New(ctx, "code", "ctxerr"))
			},
			expectedMessage:  "ctxerr_test.TestQuickWrap.func2 : ctxerr",
			expectedWarnings: 0,
			expectedCode:     "code",
		},
		{
			name: "triple wrap",
			err: func(ctx context.Context) error {
				err := errors.New("double wrap")
				err = ctxerr.QuickWrap(ctx, err)
				err = ctxerr.QuickWrap(ctx, err)
				return ctxerr.QuickWrap(ctx, err)
			},
			expectedMessage:  "ctxerr_test.TestQuickWrap.func3 : ctxerr_test.TestQuickWrap.func3 : ctxerr_test.TestQuickWrap.func3 : double wrap",
			expectedWarnings: 1,
			expectedCode:     "no_code",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			warnings := 0
			ctxerr.LogWarn = func(err error) { warnings++ }
			err := test.err(context.Background())

			if es := fmt.Sprint(err); es != test.expectedMessage {
				t.Errorf("Message did not match \n%s\n%s ", es, test.expectedMessage)
			}

			if warnings != test.expectedWarnings {
				t.Error("Warning did not match")
			}

			ce := ctxerr.Deepest(err)
			code := ce.Fields()[ctxerr.FieldKeyCode]
			if code != test.expectedCode {
				t.Error("Code did not match", code, test.expectedCode)
			}
		})
	}
}

func TestNils(t *testing.T) {
	if v := ctxerr.Deepest(nil); v != nil {
		t.Error("Unexpected non nil on Deepest of nil", v)
	}
	if v := ctxerr.Deepest(errors.New("")); v != nil {
		t.Error("Unexpected non nil on Deepest of non ctxerr", v)
	}
}

func TestDefaultOnHandle(t *testing.T) {

	tests := []struct {
		name          string
		err           error
		expectedWarn  bool
		expectedError bool
	}{
		{
			name:          "nil",
			err:           nil,
			expectedWarn:  false,
			expectedError: false,
		},
		{
			name:          "go error",
			err:           errors.New(""),
			expectedWarn:  false,
			expectedError: true,
		},
		{
			name:          "ctxerr",
			err:           ctxerr.New(context.Background(), "foo", "bar"),
			expectedWarn:  false,
			expectedError: true,
		},
		{
			name: "status code 503",
			err: func() error {
				ctx := ctxerr.SetField(context.Background(), ctxerr.FieldKeyStatusCode, 500)
				return ctxerr.New(ctx, "foo", "bar")
			}(),
			expectedWarn:  false,
			expectedError: true,
		},
		{
			name: "status code string 400",
			err: func() error {
				ctx := ctxerr.SetField(context.Background(), ctxerr.FieldKeyStatusCode, "400")
				return ctxerr.New(ctx, "foo", "bar")
			}(),
			expectedWarn:  true,
			expectedError: false,
		},
		{
			name: "status code string foo",
			err: func() error {
				ctx := ctxerr.SetField(context.Background(), ctxerr.FieldKeyStatusCode, "foo")
				return ctxerr.New(ctx, "foo", "bar")
			}(),
			expectedWarn:  false,
			expectedError: true,
		},
		{
			name: "status code 400",
			err: func() error {
				ctx := ctxerr.SetField(context.Background(), ctxerr.FieldKeyStatusCode, 400)
				return ctxerr.New(ctx, "foo", "bar")
			}(),
			expectedWarn:  true,
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logError := false
			ctxerr.LogError = func(err error) { logError = true }

			logWarn := false
			ctxerr.LogWarn = func(err error) { logWarn = true }

			ctxerr.DefaultHandler(test.err)

			if logError != test.expectedError {
				t.Error("Log severity 'error' did not match")
			}
			if logWarn != test.expectedWarn {
				t.Error("Log severity 'warn' did not match")
			}
		})
	}
}

func TestDefaultLog(t *testing.T) {
	severity := "sev"
	message := "message"
	code := "code"
	err := ctxerr.New(context.Background(), code, message)

	sb := &strings.Builder{}
	log.SetOutput(sb)

	lf := ctxerr.DefaultLog(severity)
	lf(err)

	out := strings.TrimSpace(sb.String())
	log := strings.SplitAfterN(out, " ", 3)[2]

	expectedLog := fmt.Sprintf(`%s - {"%s":"%s","%s":"%s"}`,
		message,
		ctxerr.FieldKeyCode, code,
		"error_severity", severity,
	)

	if log != expectedLog {
		t.Errorf("Logs did not match\n%s\n%s", log, expectedLog)
	}
}

func TestCallerFunc(t *testing.T) {
	cf := ctxerr.CallerFunc(0)
	expected := "ctxerr_test.TestCallerFunc"

	if cf != expected {
		t.Error("Did not match expected", cf, expected)
	}

	acf := func() string {
		return ctxerr.CallerFunc(0)
	}()
	anonExpected := "ctxerr_test.TestCallerFunc.func1"

	if acf != anonExpected {
		t.Error("Anonymous func not match expected", acf, anonExpected)
	}
}

func TestDefaultLogNonJSONFields(t *testing.T) {
	ctxerr.OnEmptyCode = func(err ctxerr.CtxErr) ctxerr.CtxErr { return err }
	ctx := ctxerr.SetField(context.Background(), "foo", func() {})
	err := ctxerr.New(ctx, "", "")

	sb := &strings.Builder{}
	log.SetOutput(sb)
	ctxerr.DefaultLog("")(err)

	out := strings.TrimSpace(sb.String())
	log := strings.SplitAfter(out, "json: ")[1]
	expectedLog := "unsupported type: func()"

	if log != expectedLog {
		t.Errorf("Logs did not match\n%s\n%s", log, expectedLog)
	}
}

func TestCategory(t *testing.T) {
	tests := []struct {
		name     string
		match    bool
		category interface{}
		toErr    func(context.Context) error
	}{
		{
			name:     "normal",
			match:    false,
			category: nil,
			toErr: func(ctx context.Context) error {
				return ctxerr.New(ctx, "code", "msg")
			},
		},
		{
			name:     "nil",
			match:    false,
			category: nil,
			toErr:    func(ctx context.Context) error { return nil },
		},
		{
			name:     "string",
			match:    true,
			category: "str",
			toErr: func(ctx context.Context) error {
				return ctxerr.New(ctx, "code", "msg")
			},
		},
		{
			name:     "int",
			match:    true,
			category: 10,
			toErr: func(ctx context.Context) error {
				return ctxerr.New(ctx, "code", "msg")
			},
		},
		{
			name:     "wrapped external",
			match:    true,
			category: "str",
			toErr: func(ctx context.Context) error {
				return ctxerr.Wrap(ctx, errors.New(""), "code", "msg")
			},
		},
		{
			name:     "wrapped has value",
			match:    true,
			category: "str",
			toErr: func(ctx context.Context) error {
				err := ctxerr.New(ctx, "code", "msg")
				return ctxerr.QuickWrap(context.Background(), err)
			},
		},
		{
			name:     "go error",
			match:    false,
			category: "str",
			toErr:    func(ctx context.Context) error { return errors.New("") },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			if test.category != nil {
				ctx = ctxerr.SetField(ctx, ctxerr.FieldKeyCategory, test.category)
			}

			err := test.toErr(ctx)
			ic := ctxerr.IsCategory(err, test.category)
			if ic != test.match {
				t.Error("Category match was unexpected")
			}
		})
	}
}

func TestAddingToContext(t *testing.T) {
	actx := context.Background()
	var bKey interface{} = "b"
	bKeyVal := "b"
	bctx := context.WithValue(actx, bKey, bKeyVal)

	if v := actx.Value(bKey); v != nil {
		t.Error("actx had a value it shouldn't have", v)

	}
	if v := bctx.Value(bKey); v != bKeyVal {
		t.Error("bctx had value that didn't match what it should have", v)
	}

	cctx := context.Background()
	dMapKey, dMapValue := "d", "d"
	dctx := ctxerr.SetField(cctx, dMapKey, dMapValue)
	eMapKey, eMapValue := "e", "e"
	ectx := ctxerr.SetField(dctx, eMapKey, eMapValue)

	if f := ctxerr.Fields(cctx); len(f) != 0 {
		t.Error("cctx had an incorrect amount of fields", f)
	}
	if f := ctxerr.Fields(dctx); len(f) != 1 {
		t.Error("dctx had an incorrect amount of fields", f)
	}
	if f := ctxerr.Fields(ectx); len(f) != 2 {
		t.Error("ectx had an incorrect amount of fields", f)
	}

	fctx := ctxerr.SetFields(ectx, map[string]interface{}{"f": "f", "g": "g"})
	if f := ctxerr.Fields(fctx); len(f) != 4 {
		t.Error("fctx had an incorrect amount of fields", f)
	}
}
