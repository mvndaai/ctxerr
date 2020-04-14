package ctxerr

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"
)

func TestFields(t *testing.T) {
	fk := "final key"
	fv := "final value"

	ctx := context.Background()
	ctx = SetField(ctx, "foo", "bar")
	ctx = SetField(ctx, fk, "baz")
	ctx = SetField(ctx, fk, fv)

	err := New(ctx, "1c678a4a-305f-4f68-880f-f459009e42ee", "msg")
	err = Wrap(ctx, err, "e6027c14-cef9-453e-965e-ff16587fecf6", "wrapper")

	f := AllFields(err)
	for k, v := range f {
		if k == fk {
			if v != fv {
				t.Error("field value was incorrect: ", v)
			}
		}
	}

	if t.Failed() {
		t.Logf("fields %+v", f)
	}
}

func TestNil(t *testing.T) {
	err := Wrap(context.Background(), nil, "", "")
	if err != nil {
		t.Error("error should have been nil")
	}
}

func TestOverall(t *testing.T) {
	code := "code"
	tests := []struct {
		name  string
		toErr func(context.Context) error

		expectedMessage string
		expectedFields  map[string]interface{}
		expectedNil     bool
	}{
		{
			name:        "nil",
			toErr:       func(ctx context.Context) error { return nil },
			expectedNil: true,
		},
		{
			name:            "errors",
			toErr:           func(ctx context.Context) error { return errors.New("errors") },
			expectedMessage: "errors",
			expectedFields:  map[string]interface{}{},
		},
		{
			name: "new",
			toErr: func(ctx context.Context) error {
				return New(ctx, code, "", "new")
			},
			expectedMessage: "new",
			expectedFields:  map[string]interface{}{FieldKeyCode: code},
		},
		{
			name: "newf",
			toErr: func(ctx context.Context) error {
				return Newf(ctx, code, "%s", "newf")
			},
			expectedMessage: "newf",
			expectedFields:  map[string]interface{}{FieldKeyCode: code},
		},
		{
			name: "action",
			toErr: func(ctx context.Context) error {
				ctx = SetField(ctx, FieldKeyAction, "action")
				return New(ctx, code, "action")
			},
			expectedMessage: "action",
			expectedFields: map[string]interface{}{
				FieldKeyCode:   code,
				FieldKeyAction: "action",
			},
		},
		{
			name: "wrap",
			toErr: func(ctx context.Context) error {
				return Wrap(ctx, errors.New("wrapped"), code, "", "wrap")
			},
			expectedMessage: "wrap : wrapped",
			expectedFields:  map[string]interface{}{FieldKeyCode: code},
		},
		{
			name: "wrapf",
			toErr: func(ctx context.Context) error {
				return Wrapf(ctx, errors.New("wrapped"), code, "%s", "wrapf")
			},
			expectedMessage: "wrapf : wrapped",
			expectedFields:  map[string]interface{}{FieldKeyCode: code},
		},
		{
			name: "nil wrap",
			toErr: func(ctx context.Context) error {
				return Wrap(ctx, nil, code, "nil wrap")
			},
			expectedNil: true,
		},
		{
			name: "nil wrapf",
			toErr: func(ctx context.Context) error {
				return Wrapf(ctx, nil, code, "nil wrap")
			},
			expectedNil: true,
		},
		{
			name: "with fields",
			toErr: func(ctx context.Context) error {
				ctx = SetFields(ctx, map[string]interface{}{"foo": "bar"})
				return New(ctx, code, "with fields")
			},
			expectedMessage: "with fields",
			expectedFields:  map[string]interface{}{FieldKeyCode: code, "foo": "bar"},
		},
		{
			name: "new no code",
			toErr: func(ctx context.Context) error {
				return New(ctx, "", "new no code")
			},
			expectedMessage: "new no code",
			expectedFields:  map[string]interface{}{},
		},
		{
			name: "newf no code",
			toErr: func(ctx context.Context) error {
				return Newf(ctx, "", "new no code")
			},
			expectedMessage: "new no code",
			expectedFields:  map[string]interface{}{},
		},
		{
			name: "wrapf no code",
			toErr: func(ctx context.Context) error {
				return Wrapf(ctx, errors.New("error"), "", "wrapf no code")
			},
			expectedMessage: "wrapf no code : error",
			expectedFields:  map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := tt.toErr(context.Background())
			if err != nil {
				if m := err.Error(); m != tt.expectedMessage {
					t.Errorf("Message did not match log message:\n%s\n%s", m, tt.expectedMessage)
				}
			}

			if (err != nil) == tt.expectedNil {
				t.Error("Nil matched failed")
			}

			logFields := AllFields(err)
			if len(tt.expectedFields) != len(logFields) {
				t.Errorf("Fields count did not match:\n%s\n%s", logFields, tt.expectedFields)

			}
			for k, v := range tt.expectedFields {
				if fv := logFields[k]; fv != v {
					t.Error("field did not match", fv, k)
				}
			}
		})

	}
}

func TestQuickWrap(t *testing.T) {
	tests := []struct {
		name string
		err  func(context.Context) error

		expectedMessage string
		expectedCode    interface{}
	}{
		{
			name:            "external",
			err:             func(ctx context.Context) error { return QuickWrap(ctx, errors.New("external")) },
			expectedMessage: "ctxerr.TestQuickWrap.func1 : external",
			expectedCode:    nil,
		},
		{
			name: "ctxerr",
			err: func(ctx context.Context) error {
				return QuickWrap(ctx, New(ctx, "code", "ctxerr"))
			},
			expectedMessage: "ctxerr.TestQuickWrap.func2 : ctxerr",
			expectedCode:    "code",
		},
		{
			name: "triple wrap",
			err: func(ctx context.Context) error {
				err := errors.New("double wrap")
				err = QuickWrap(ctx, err)
				err = QuickWrap(ctx, err)
				return QuickWrap(ctx, err)
			},
			expectedMessage: "ctxerr.TestQuickWrap.func3 : ctxerr.TestQuickWrap.func3 : ctxerr.TestQuickWrap.func3 : double wrap",
			expectedCode:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.err(context.Background())

			if es := fmt.Sprint(err); es != test.expectedMessage {
				t.Errorf("Message did not match \n%s\n%s ", es, test.expectedMessage)
			}

			code := AllFields(err)[FieldKeyCode]
			if code != test.expectedCode {
				t.Error("Code did not match", code, test.expectedCode)
			}
		})
	}
}

func TestCallerFunc(t *testing.T) {
	cf := CallerFunc(0)
	expected := "ctxerr.TestCallerFunc"

	if cf != expected {
		t.Error("Did not match expected", cf, expected)
	}

	acf := func() string {
		return CallerFunc(0)
	}()
	anonExpected := "ctxerr.TestCallerFunc.func1"

	if acf != anonExpected {
		t.Error("Anonymous func not match expected", acf, anonExpected)
	}
}

func TestDefaultLogNonJSONFields(t *testing.T) {
	ctx := SetField(context.Background(), "foo", func() {})
	err := New(ctx, "", "")

	sb := &strings.Builder{}
	log.SetOutput(sb)
	DefaultLogHook(err)

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
				return New(ctx, "code", "msg")
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
				return New(ctx, "code", "msg")
			},
		},
		{
			name:     "int",
			match:    true,
			category: 10,
			toErr: func(ctx context.Context) error {
				return New(ctx, "code", "msg")
			},
		},
		{
			name:     "wrapped external",
			match:    true,
			category: "str",
			toErr: func(ctx context.Context) error {
				return Wrap(ctx, errors.New(""), "code", "msg")
			},
		},
		{
			name:     "wrapped has value",
			match:    true,
			category: "str",
			toErr: func(ctx context.Context) error {
				err := New(ctx, "code", "msg")
				return QuickWrap(context.Background(), err)
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
				ctx = SetField(ctx, FieldKeyCategory, test.category)
			}

			err := test.toErr(ctx)
			ic := HasCategory(err, test.category)
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
	dctx := SetField(cctx, dMapKey, dMapValue)
	eMapKey, eMapValue := "e", "e"
	ectx := SetField(dctx, eMapKey, eMapValue)

	if f := Fields(cctx); len(f) != 0 {
		t.Error("cctx had an incorrect amount of fields", f)
	}
	if f := Fields(dctx); len(f) != 1 {
		t.Error("dctx had an incorrect amount of fields", f)
	}
	if f := Fields(ectx); len(f) != 2 {
		t.Error("ectx had an incorrect amount of fields", f)
	}

	fctx := SetFields(ectx, map[string]interface{}{"f": "f", "g": "g"})
	if f := Fields(fctx); len(f) != 4 {
		t.Error("fctx had an incorrect amount of fields", f)
	}
}

func TestAllFields(t *testing.T) {
	if f := AllFields(nil); f == nil {
		t.Error("fields shouldn't have been nil")
	}
	if f := AllFields(errors.New("a")); len(f) != 0 {
		t.Error("fields without a context should have no values")
	}

	ctx := context.Background()
	ctx = SetField(ctx, "a", "a")
	err := New(ctx, "a", "a")

	ctx = SetField(ctx, "b", "b")
	err = QuickWrap(ctx, err)

	fields := AllFields(err)
	b, ok := fields["b"]
	if !ok {
		t.Fatal("b was not found in fields")
	}
	if b != "b" {
		t.Error("b value did not match", b)
	}
}

func TestAs(t *testing.T) {
	tests := []struct {
		in  error
		out bool
	}{
		{in: nil, out: false},
		{in: errors.New("errors"), out: false},
		{in: New(context.Background(), "-", "ctxerr"), out: true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintln(tt.in), func(t *testing.T) {
			if _, b := As(tt.in); b != tt.out {
				t.Error("bool did not match")
			}
		})
	}

}

func TestFallbackHandle(t *testing.T) {
	tests := []struct {
		name        string
		HandleHooks []func(error)
		expectedLog bool
	}{
		{name: "fallback",
			HandleHooks: nil,
			expectedLog: true,
		},
		{name: "normal",
			HandleHooks: []func(error){func(error) {}},
			expectedLog: false,
		},
	}

	err := New(context.Background(), "", "msg")
	sb := &strings.Builder{}
	log.SetOutput(sb)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb.Reset()
			handleHooks = tt.HandleHooks
			Handle(err)

			msg := sb.String()
			if (len(sb.String()) == 0) == tt.expectedLog {
				t.Error("expectedLog did not match:", msg)
			}
		})
	}
}

func TestCustomizations(t *testing.T) {
	var createHookRan bool
	var handleHookRan bool

	AddCreateHook(func(ctx context.Context, _ string, _ error) context.Context {
		createHookRan = true
		return ctx
	})

	AddHandleHook(func(_ error) {
		handleHookRan = true
	})

	if createHookRan {
		t.Error("create hook should not have run yet")
	}
	err := New(context.Background(), "", "")
	if !createHookRan {
		t.Error("did not run create hook")
	}

	if handleHookRan {
		t.Error("handle hook should not have run yet")
	}
	Handle(err)
	if !handleHookRan {
		t.Error("did not run handle hook")
	}

}

func TestShortcutFunctions(t *testing.T) {
	ctx := context.Background()

	code := http.StatusOK
	ctx = SetHTTPStatusCode(ctx, code)
	if v := Fields(ctx)[FieldKeyStatusCode]; v != code {
		t.Error("SetHTTPStatusCode did work as expected: ", v)
	}

	action := "action"
	ctx = SetAction(ctx, action)
	if v := Fields(ctx)[FieldKeyAction]; v != action {
		t.Error("SetAction did work as expected: ", v)
	}

	category := "category"
	ctx = SetCategory(ctx, category)
	if v := Fields(ctx)[FieldKeyCategory]; v != category {
		t.Error("SetCategory did work as expected: ", v)
	}
}
