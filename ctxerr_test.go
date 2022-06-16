package ctxerr

import (
	"context"
	"encoding/json"
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

			// Ignore location field
			delete(logFields, FieldKeyLocation)

			t.Logf("fields %+v", logFields)
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
			expectedMessage: "external",
			expectedCode:    nil,
		},
		{
			name: "ctxerr",
			err: func(ctx context.Context) error {
				return QuickWrap(ctx, New(ctx, "code", "ctxerr"))
			},
			expectedMessage: "ctxerr",
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
			expectedMessage: "double wrap",
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

func TestLocation(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		locationPrefixs []string // this accounts for annonymous functions
	}{
		{
			name:            "New",
			err:             New(context.Background(), "code", nil),
			locationPrefixs: []string{"ctxerr.TestLocation"},
		},
		{
			name:            "Newf",
			err:             Newf(context.Background(), "code", ""),
			locationPrefixs: []string{"ctxerr.TestLocation"},
		},
		{
			name:            "Wrap",
			err:             Wrap(context.Background(), fmt.Errorf(""), "code", ""),
			locationPrefixs: []string{"ctxerr.TestLocation"},
		},
		{
			name:            "Wrapf",
			err:             Wrapf(context.Background(), fmt.Errorf(""), "code", ""),
			locationPrefixs: []string{"ctxerr.TestLocation"},
		},
		{
			name:            "NewHTTP",
			err:             NewHTTP(context.Background(), "code", "", 0, nil),
			locationPrefixs: []string{"ctxerr.TestLocation"},
		},
		{
			name:            "NewHTTPf",
			err:             NewHTTPf(context.Background(), "code", "", 0, ""),
			locationPrefixs: []string{"ctxerr.TestLocation"},
		},
		{
			name:            "WrapHTTP",
			err:             WrapHTTP(context.Background(), fmt.Errorf(""), "code", "", 0, nil),
			locationPrefixs: []string{"ctxerr.TestLocation"},
		},
		{
			name:            "WrapHTTPf",
			err:             WrapHTTPf(context.Background(), fmt.Errorf(""), "code", "", 0, ""),
			locationPrefixs: []string{"ctxerr.TestLocation"},
		},
		{
			name:            "QuickWrap",
			err:             QuickWrap(context.Background(), fmt.Errorf("")),
			locationPrefixs: []string{"ctxerr.TestLocation"},
		},
		{
			name: "complicated",
			err: func() error {
				ctx := context.Background()
				err := func(ctx context.Context) error {
					return New(ctx, "code", "")
				}(ctx)
				return QuickWrap(ctx, err)
			}(),
			locationPrefixs: []string{"ctxerr.TestLocation.func1", "ctxerr.TestLocation.func1.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locs := AllFields(tt.err)[FieldKeyLocation].([]interface{})
			if len(locs) == 0 {
				t.Error("Location count is 0")
			}

			if len(locs) != len(tt.locationPrefixs) {
				b1, _ := json.Marshal(locs)
				b2, _ := json.Marshal(tt.locationPrefixs)
				t.Logf("Locations :\n%s\n%s", string(b1), string(b2))
				t.Errorf("Location count did not match:%v -%v", len(locs), len(tt.locationPrefixs))
			}

			for i, loc := range locs {
				t.Logf("'%s' '%s'", fmt.Sprint(loc), tt.locationPrefixs[i])
				if !strings.HasPrefix(fmt.Sprint(loc), tt.locationPrefixs[i]) {
					t.Errorf("Location did not match:\n'%s'\n'%s'", loc, tt.locationPrefixs[i])
				}
			}
		})
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

type testContextKey string

func TestAddingToContext(t *testing.T) {
	actx := context.Background()
	var bKey testContextKey = "b"
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

func TestWrappingWithNoUnderlyingCode(t *testing.T) {
	expectedCode := "expected"
	ctx := context.Background()
	err := New(ctx, "", "0")
	err = Wrap(ctx, err, expectedCode, "1")

	code := AllFields(err)[FieldKeyCode]
	if code != expectedCode {
		t.Error("code did not match", code)
	}
}

func TestHTTPFuncs(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		expectedCode       interface{}
		expectedAction     interface{}
		expectedStatusCode interface{}
		expectedMessage    string
	}{
		{
			name:               "NewHTTP",
			err:                NewHTTP(context.Background(), "c", "a", http.StatusBadRequest, "m", 1, "v"),
			expectedCode:       "c",
			expectedAction:     "a",
			expectedStatusCode: http.StatusBadRequest,
			expectedMessage:    "m1v",
		},
		{
			name:               "NewHTTPf",
			err:                NewHTTPf(context.Background(), "c", "a", http.StatusConflict, "m %02d%s", 1, "v"),
			expectedCode:       "c",
			expectedAction:     "a",
			expectedStatusCode: http.StatusConflict,
			expectedMessage:    "m 01v",
		},
		{
			name:               "WrapHTTP",
			err:                WrapHTTP(context.Background(), fmt.Errorf("e"), "c", "a", 0, "m", 1, "v"),
			expectedCode:       "c",
			expectedAction:     "a",
			expectedStatusCode: nil,
			expectedMessage:    "m1v : e",
		},
		{
			name:               "WrapHTTPf",
			err:                WrapHTTPf(context.Background(), fmt.Errorf("e"), "c", "a", http.StatusBadRequest, "m %02d%s", 1, "v"),
			expectedCode:       "c",
			expectedAction:     "a",
			expectedStatusCode: http.StatusBadRequest,
			expectedMessage:    "m 01v : e",
		},
		{
			name:               "WrapHTTP nil error",
			err:                WrapHTTP(context.Background(), nil, "c", "a", 0, "m", 1, "v"),
			expectedCode:       nil,
			expectedAction:     nil,
			expectedStatusCode: nil,
			expectedMessage:    "ignored",
		},
		{
			name:               "WrapHTTPf nil error",
			err:                WrapHTTPf(context.Background(), nil, "c", "a", http.StatusBadRequest, "m %02d%s", 1, "v"),
			expectedCode:       nil,
			expectedAction:     nil,
			expectedStatusCode: nil,
			expectedMessage:    "ignored",
		},
		{
			name: "Status-code-already-on-context",
			err: func() error {
				ctx := SetHTTPStatusCode(context.Background(), http.StatusInternalServerError)
				return NewHTTP(ctx, "c", "a", http.StatusBadRequest, "m")
			}(),
			expectedCode:       "c",
			expectedAction:     "a",
			expectedStatusCode: http.StatusBadRequest,
			expectedMessage:    "m",
		},
		{
			name: "Wrapped error already has status code and action",
			err: func() error {
				err := NewHTTP(context.Background(), "ci", "ai", http.StatusBadRequest, "mi")
				return WrapHTTP(context.Background(), err, "co", "ao", http.StatusConflict, "mo")
			}(),
			expectedCode:       "ci",
			expectedAction:     "ai",
			expectedStatusCode: http.StatusBadRequest,
			expectedMessage:    "mo : mi",
		},
		{
			name: "Wrapped error already has status code and action same context",
			err: func() error {
				ctx := context.Background()
				err := NewHTTP(ctx, "ci", "ai", http.StatusBadRequest, "mi")
				return WrapHTTP(ctx, err, "co", "ao", http.StatusConflict, "mo")
			}(),
			expectedCode:       "ci",
			expectedAction:     "ai",
			expectedStatusCode: http.StatusBadRequest,
			expectedMessage:    "mo : mi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := AllFields(tt.err)

			if v := f[FieldKeyCode]; v != tt.expectedCode {
				t.Errorf("code did not match: %v", v)
			}
			if v := f[FieldKeyAction]; v != tt.expectedAction {
				t.Errorf("action did not match: %v - %v", v, tt.expectedAction)
			}
			if v := f[FieldKeyStatusCode]; v != tt.expectedStatusCode {
				t.Errorf("status code did not match: %v - %v", v, tt.expectedStatusCode)
			}
			if tt.err != nil {
				if v := tt.err.Error(); v != tt.expectedMessage {
					t.Errorf("message did not match: %v - %v", v, tt.expectedMessage)
				}
			}
		})
	}
}
