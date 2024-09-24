package ctxerr_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
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

	err := ctxerr.New(ctx, "1c678a4a-305f-4f68-880f-f459009e42ee", "msg")
	err = ctxerr.Wrap(ctx, err, "e6027c14-cef9-453e-965e-ff16587fecf6", "wrapper")

	f := ctxerr.AllFields(err)
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
	err := ctxerr.Wrap(context.Background(), nil, "", "")
	if err != nil {
		t.Error("error should have been nil")
	}
	ctxerr.Handle(err)

	// Prettier message when instance is nil
	func() {
		defer func() {
			if r := recover(); r != nil {
				if !strings.HasSuffix(fmt.Sprint(r), "ctxerr.Instance is nil") {
					t.Error("recovered with wrong message:", r)
				}
			} else {
				t.Error("expected to recover")
			}
		}()
		var in *ctxerr.Instance
		in.AddCreateHook(ctxerr.SetCodeHook)
	}()

	func() {
		defer func() {
			if r := recover(); r != nil {
				if !strings.HasSuffix(fmt.Sprint(r), "ctxerr.Instance is nil") {
					t.Error("recovered with wrong message:", r)
				}
			} else {
				t.Error("expected to recover")
			}
		}()
		var in *ctxerr.Instance
		in.AddHandleHook(ctxerr.DefaultLogHook)
	}()

	func() {
		defer func() {
			if r := recover(); r != nil {
				if !strings.HasSuffix(fmt.Sprint(r), "ctxerr.Instance is nil") {
					t.Error("recovered with wrong message:", r)
				}
			} else {
				t.Error("expected to recover")
			}
		}()
		var in *ctxerr.Instance
		in.AddFieldHook(func(_ context.Context, v any) any { return v })
	}()

	func() {
		defer func() {
			if r := recover(); r != nil {
				if !strings.HasSuffix(fmt.Sprint(r), "ctxerr.Instance is nil") {
					t.Error("recovered with wrong message:", r)
				}
			} else {
				t.Error("expected to recover")
			}
		}()
		var in *ctxerr.Instance
		in.AddFieldsFunc(func(error) map[string]any { return nil })
	}()
}

func TestOverall(t *testing.T) {
	code := "code"
	tests := []struct {
		name  string
		toErr func(context.Context) error

		expectedMessage string
		expectedFields  map[string]any
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
			expectedFields:  map[string]any{},
		},
		{
			name: "new",
			toErr: func(ctx context.Context) error {
				return ctxerr.New(ctx, code, "", "new")
			},
			expectedMessage: "new",
			expectedFields:  map[string]any{ctxerr.FieldKeyCode: code},
		},
		{
			name: "newf",
			toErr: func(ctx context.Context) error {
				return ctxerr.Newf(ctx, code, "%s", "newf")
			},
			expectedMessage: "newf",
			expectedFields:  map[string]any{ctxerr.FieldKeyCode: code},
		},
		{
			name: "action",
			toErr: func(ctx context.Context) error {
				ctx = ctxerr.SetField(ctx, ctxerr.FieldKeyAction, "action")
				return ctxerr.New(ctx, code, "action")
			},
			expectedMessage: "action",
			expectedFields: map[string]any{
				ctxerr.FieldKeyCode:   code,
				ctxerr.FieldKeyAction: "action",
			},
		},
		{
			name: "wrap",
			toErr: func(ctx context.Context) error {
				return ctxerr.Wrap(ctx, errors.New("wrapped"), code, "", "wrap")
			},
			expectedMessage: "wrap : wrapped",
			expectedFields:  map[string]any{ctxerr.FieldKeyCode: code},
		},
		{
			name: "wrapf",
			toErr: func(ctx context.Context) error {
				return ctxerr.Wrapf(ctx, errors.New("wrapped"), code, "%s", "wrapf")
			},
			expectedMessage: "wrapf : wrapped",
			expectedFields:  map[string]any{ctxerr.FieldKeyCode: code},
		},
		{
			name: "nil wrap",
			toErr: func(ctx context.Context) error {
				return ctxerr.Wrap(ctx, nil, code, "nil wrap")
			},
			expectedNil: true,
		},
		{
			name: "nil wrapf",
			toErr: func(ctx context.Context) error {
				return ctxerr.Wrapf(ctx, nil, code, "nil wrap")
			},
			expectedNil: true,
		},
		{
			name: "with fields",
			toErr: func(ctx context.Context) error {
				ctx = ctxerr.SetFields(ctx, map[string]any{"foo": "bar"})
				return ctxerr.New(ctx, code, "with fields")
			},
			expectedMessage: "with fields",
			expectedFields:  map[string]any{ctxerr.FieldKeyCode: code, "foo": "bar"},
		},
		{
			name: "new no code",
			toErr: func(ctx context.Context) error {
				return ctxerr.New(ctx, "", "new no code")
			},
			expectedMessage: "new no code",
			expectedFields:  map[string]any{},
		},
		{
			name: "newf no code",
			toErr: func(ctx context.Context) error {
				return ctxerr.Newf(ctx, "", "new no code")
			},
			expectedMessage: "new no code",
			expectedFields:  map[string]any{},
		},
		{
			name: "wrapf no code",
			toErr: func(ctx context.Context) error {
				return ctxerr.Wrapf(ctx, errors.New("error"), "", "wrapf no code")
			},
			expectedMessage: "wrapf no code : error",
			expectedFields:  map[string]any{},
		},
		{
			name: "wrap no message",
			toErr: func(ctx context.Context) error {
				return ctxerr.Wrap(ctx, errors.New("error"), "")
			},
			expectedMessage: "error",
			expectedFields:  map[string]any{},
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

			logFields := ctxerr.AllFields(err)

			// Ignore location field
			delete(logFields, ctxerr.FieldKeyLocation)

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
		expectedCode    any
	}{
		{
			name:            "external",
			err:             func(ctx context.Context) error { return ctxerr.QuickWrap(ctx, errors.New("external")) },
			expectedMessage: "external",
			expectedCode:    nil,
		},
		{
			name: "ctxerr",
			err: func(ctx context.Context) error {
				return ctxerr.QuickWrap(ctx, ctxerr.New(ctx, "code", "ctxerr"))
			},
			expectedMessage: "ctxerr",
			expectedCode:    "code",
		}, {
			name: "ctxerr instance",
			err: func(ctx context.Context) error {
				in := ctxerr.NewInstance()
				return in.QuickWrap(ctx, ctxerr.New(ctx, "code", "ctxerr"))
			},
			expectedMessage: "ctxerr",
			expectedCode:    "code",
		},
		{
			name: "triple wrap",
			err: func(ctx context.Context) error {
				err := errors.New("double wrap")
				err = ctxerr.QuickWrap(ctx, err)
				err = ctxerr.QuickWrap(ctx, err)
				return ctxerr.QuickWrap(ctx, err)
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

			code := ctxerr.AllFields(err)[ctxerr.FieldKeyCode]
			if code != test.expectedCode {
				t.Error("Code did not match", code, test.expectedCode)
			}
		})
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

func TestLocation(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		locationPrefixs []string // this accounts for annonymous functions
	}{
		{
			name:            "New",
			err:             ctxerr.New(context.Background(), "code", nil),
			locationPrefixs: []string{"ctxerr_test.TestLocation"},
		},
		{
			name:            "Newf",
			err:             ctxerr.Newf(context.Background(), "code", ""),
			locationPrefixs: []string{"ctxerr_test.TestLocation"},
		},
		{
			name:            "Wrap",
			err:             ctxerr.Wrap(context.Background(), fmt.Errorf(""), "code", ""),
			locationPrefixs: []string{"ctxerr_test.TestLocation"},
		},
		{
			name:            "Wrapf",
			err:             ctxerr.Wrapf(context.Background(), fmt.Errorf(""), "code", ""),
			locationPrefixs: []string{"ctxerr_test.TestLocation"},
		},
		{
			name:            "NewHTTP",
			err:             ctxerr.NewHTTP(context.Background(), "code", "", 0, nil),
			locationPrefixs: []string{"ctxerr_test.TestLocation"},
		},
		{
			name:            "NewHTTPf",
			err:             ctxerr.NewHTTPf(context.Background(), "code", "", 0, ""),
			locationPrefixs: []string{"ctxerr_test.TestLocation"},
		},
		{
			name:            "WrapHTTP",
			err:             ctxerr.WrapHTTP(context.Background(), fmt.Errorf(""), "code", "", 0, nil),
			locationPrefixs: []string{"ctxerr_test.TestLocation"},
		},
		{
			name:            "WrapHTTPf",
			err:             ctxerr.WrapHTTPf(context.Background(), fmt.Errorf(""), "code", "", 0, ""),
			locationPrefixs: []string{"ctxerr_test.TestLocation"},
		},
		{
			name:            "QuickWrap",
			err:             ctxerr.QuickWrap(context.Background(), fmt.Errorf("")),
			locationPrefixs: []string{"ctxerr_test.TestLocation"},
		},
		{
			name: "complicated",
			err: func() error {
				ctx := context.Background()
				err := func(ctx context.Context) error {
					return ctxerr.New(ctx, "code", "")
				}(ctx)
				return ctxerr.QuickWrap(ctx, err)
			}(),
			locationPrefixs: []string{"ctxerr_test.TestLocation.func1", "ctxerr_test.TestLocation.func1.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locs := ctxerr.AllFields(tt.err)[ctxerr.FieldKeyLocation].([]any)
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
	ctx := ctxerr.SetField(context.Background(), "foo", func() {})
	err := ctxerr.New(ctx, "CODE", "msg")

	// Capture stdout
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the test
	ctxerr.DefaultLogHook(err)

	// Set stdout back to normal; read the output
	w.Close()
	os.Stdout = originalStdout
	out, _ := io.ReadAll(r)

	expectedFields := map[string]any{
		//"time":           "2024-03-06T14:33:09.03475-07:00",
		"level":          "ERROR",
		"msg":            "msg",
		"foo":            "!ERROR:json: unsupported type: func()",
		"error_code":     "CODE",
		"error_location": []any{"ctxerr_test.TestDefaultLogNonJSONFields"},
	}

	actualMap := map[string]any{}
	err = json.Unmarshal([]byte(out), &actualMap)
	if err != nil {
		t.Error(err)
	}
	for k, v := range expectedFields {
		if av, ok := actualMap[k]; !ok || fmt.Sprint(av) != fmt.Sprint(v) {
			t.Errorf("Field did not match\n%s\n%s", av, v)
		}
	}

	if _, ok := actualMap["time"]; !ok {
		t.Error("time field was not present")
	}
}

func TestCategory(t *testing.T) {
	tests := []struct {
		name     string
		match    bool
		category any
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
			ic := ctxerr.HasCategory(err, test.category)
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

	fctx := ctxerr.SetFields(ectx, map[string]any{"f": "f", "g": "g"})
	if f := ctxerr.Fields(fctx); len(f) != 4 {
		t.Error("fctx had an incorrect amount of fields", f)
	}
}

func TestAllFields(t *testing.T) {
	if f := ctxerr.AllFields(nil); f == nil {
		t.Error("fields shouldn't have been nil")
	}
	if f := ctxerr.AllFields(errors.New("a")); len(f) != 0 {
		t.Error("fields without a context should have no values")
	}

	ctx := context.Background()
	ctx = ctxerr.SetField(ctx, "a", "a")
	err := ctxerr.New(ctx, "a", "a")

	ctx = ctxerr.SetField(ctx, "b", "b")
	err = ctxerr.QuickWrap(ctx, err)

	fields := ctxerr.AllFields(err)
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
		{in: ctxerr.New(context.Background(), "-", "ctxerr"), out: true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintln(tt.in), func(t *testing.T) {
			if _, b := ctxerr.As(tt.in); b != tt.out {
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

	err := ctxerr.New(context.Background(), "", "msg")
	sb := &strings.Builder{}
	log.SetOutput(sb)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb.Reset()
			in := ctxerr.NewInstance()
			in.HandleHooks = tt.HandleHooks
			in.Handle(err)

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

	ctxerr.AddCreateHook(func(ctx context.Context, _ string, _ error) context.Context {
		createHookRan = true
		return ctx
	})

	ctxerr.AddHandleHook(func(_ error) {
		handleHookRan = true
	})

	if createHookRan {
		t.Error("create hook should not have run yet")
	}
	err := ctxerr.New(context.Background(), "", "")
	if !createHookRan {
		t.Error("did not run create hook")
	}

	if handleHookRan {
		t.Error("handle hook should not have run yet")
	}
	ctxerr.Handle(err)
	if !handleHookRan {
		t.Error("did not run handle hook")
	}

}

func TestShortcutFunctions(t *testing.T) {
	ctx := context.Background()

	code := http.StatusOK
	ctx = ctxerr.SetHTTPStatusCode(ctx, code)
	if v := ctxerr.Fields(ctx)[ctxerr.FieldKeyStatusCode]; v != code {
		t.Error("SetHTTPStatusCode did work as expected: ", v)
	}

	action := "action"
	ctx = ctxerr.SetAction(ctx, action)
	if v := ctxerr.Fields(ctx)[ctxerr.FieldKeyAction]; v != action {
		t.Error("SetAction did work as expected: ", v)
	}

	category := "category"
	ctx = ctxerr.SetCategory(ctx, category)
	if v := ctxerr.Fields(ctx)[ctxerr.FieldKeyCategory]; v != category {
		t.Error("SetCategory did work as expected: ", v)
	}
}

func TestWrappingWithNoUnderlyingCode(t *testing.T) {
	expectedCode := "expected"
	ctx := context.Background()
	err := ctxerr.New(ctx, "", "0")
	err = ctxerr.Wrap(ctx, err, expectedCode, "1")

	code := ctxerr.AllFields(err)[ctxerr.FieldKeyCode]
	if code != expectedCode {
		t.Error("code did not match", code)
	}
}

func TestHTTPFuncs(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		expectedCode       any
		expectedAction     any
		expectedStatusCode any
		expectedMessage    string
	}{
		{
			name:               "NewHTTP",
			err:                ctxerr.NewHTTP(context.Background(), "c", "a", http.StatusBadRequest, "m", 1, "v"),
			expectedCode:       "c",
			expectedAction:     "a",
			expectedStatusCode: http.StatusBadRequest,
			expectedMessage:    "m1v",
		},
		{
			name:               "NewHTTPf",
			err:                ctxerr.NewHTTPf(context.Background(), "c", "a", http.StatusConflict, "m %02d%s", 1, "v"),
			expectedCode:       "c",
			expectedAction:     "a",
			expectedStatusCode: http.StatusConflict,
			expectedMessage:    "m 01v",
		},
		{
			name:               "WrapHTTP",
			err:                ctxerr.WrapHTTP(context.Background(), fmt.Errorf("e"), "c", "a", 0, "m", 1, "v"),
			expectedCode:       "c",
			expectedAction:     "a",
			expectedStatusCode: nil,
			expectedMessage:    "m1v : e",
		},
		{
			name:               "WrapHTTPf",
			err:                ctxerr.WrapHTTPf(context.Background(), fmt.Errorf("e"), "c", "a", http.StatusBadRequest, "m %02d%s", 1, "v"),
			expectedCode:       "c",
			expectedAction:     "a",
			expectedStatusCode: http.StatusBadRequest,
			expectedMessage:    "m 01v : e",
		},
		{
			name:               "WrapHTTP nil error",
			err:                ctxerr.WrapHTTP(context.Background(), nil, "c", "a", 0, "m", 1, "v"),
			expectedCode:       nil,
			expectedAction:     nil,
			expectedStatusCode: nil,
			expectedMessage:    "ignored",
		},
		{
			name:               "WrapHTTPf nil error",
			err:                ctxerr.WrapHTTPf(context.Background(), nil, "c", "a", http.StatusBadRequest, "m %02d%s", 1, "v"),
			expectedCode:       nil,
			expectedAction:     nil,
			expectedStatusCode: nil,
			expectedMessage:    "ignored",
		},
		{
			name: "Status-code-already-on-context",
			err: func() error {
				ctx := ctxerr.SetHTTPStatusCode(context.Background(), http.StatusInternalServerError)
				return ctxerr.NewHTTP(ctx, "c", "a", http.StatusBadRequest, "m")
			}(),
			expectedCode:       "c",
			expectedAction:     "a",
			expectedStatusCode: http.StatusBadRequest,
			expectedMessage:    "m",
		},
		{
			name: "Wrapped error already has status code and action",
			err: func() error {
				err := ctxerr.NewHTTP(context.Background(), "ci", "ai", http.StatusBadRequest, "mi")
				return ctxerr.WrapHTTP(context.Background(), err, "co", "ao", http.StatusConflict, "mo")
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
				err := ctxerr.NewHTTP(ctx, "ci", "ai", http.StatusBadRequest, "mi")
				return ctxerr.WrapHTTP(ctx, err, "co", "ao", http.StatusConflict, "mo")
			}(),
			expectedCode:       "ci",
			expectedAction:     "ai",
			expectedStatusCode: http.StatusBadRequest,
			expectedMessage:    "mo : mi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := ctxerr.AllFields(tt.err)

			if v := f[ctxerr.FieldKeyCode]; v != tt.expectedCode {
				t.Errorf("code did not match: %v", v)
			}
			if v := f[ctxerr.FieldKeyAction]; v != tt.expectedAction {
				t.Errorf("action did not match: %v - %v", v, tt.expectedAction)
			}
			if v := f[ctxerr.FieldKeyStatusCode]; v != tt.expectedStatusCode {
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

func TestHasField(t *testing.T) {
	ctx := context.Background()
	err := ctxerr.New(ctx, "c")

	if !ctxerr.HasField(err, ctxerr.FieldKeyCode) {
		t.Error("expected code field", ctxerr.FieldKeyCode, ctxerr.AllFields(err))
	}

	key := "key"
	if ctxerr.HasField(err, key) {
		t.Error("expected no field", key, ctxerr.AllFields(err))
	}

	ctx = ctxerr.SetField(ctx, key, "")
	err = ctxerr.Wrap(context.Background(), ctxerr.Wrap(ctx, fmt.Errorf(""), "c"), "c")

	if !ctxerr.HasField(err, key) {
		t.Error("expected field", key, ctxerr.AllFields(err))
	}

	err = fmt.Errorf("abc")
	if ctxerr.HasField(err, ctxerr.FieldKeyCode) {
		t.Error("expected no code field", ctxerr.FieldKeyCode, ctxerr.AllFields(err))
	}
}

func TestImpossibilties(t *testing.T) {
	// Main field key is not the a map
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxerr.FieldsKey, "s")
	f := ctxerr.Fields(ctx)
	if f != nil {
		t.Error("expected fields to be nil")
	}
}

func TestImpl(t *testing.T) {
	var e1 ctxerr.CtxErr = ctxerr.New(context.Background(), "c1").(ctxerr.CtxErr)
	var e2 ctxerr.CtxErr = ctxerr.New(context.Background(), "c2").(ctxerr.CtxErr)
	if !e1.Is(e2) {
		t.Error("expected is to work")
	}

	ctx := e1.Context()
	if code := ctxerr.Fields(ctx)[ctxerr.FieldKeyCode]; code == "" {
		t.Error("expected a code")
	}
	e1.WithContext(context.Background())
	if code := ctxerr.Fields(ctx)[ctxerr.FieldKeyCode]; code == "" {
		t.Error("expected no code got", code)
	}
}

func TestFeildsWithNilCtx(t *testing.T) {
	var ctx context.Context
	f := ctxerr.Fields(ctx)
	if f != nil {
		t.Error("expected a nil map")
	}
}

type redactable string

func (r redactable) Redact() any {
	return "redacted"
}

type IRedactable interface {
	Redact() any
}

func RedactItem(ctx context.Context, a any) any {
	if v, ok := a.(IRedactable); ok {
		return v.Redact()
	}
	return a
}

func TestFieldHook(t *testing.T) {
	in := ctxerr.Instance{}
	in.AddFieldHook(RedactItem)

	var key = "key"
	tests := []struct {
		name string
		f    func(context.Context, any) context.Context
	}{
		{
			name: "in.SetField",
			f: func(ctx context.Context, v any) context.Context {
				return in.SetField(ctx, key, v)
			},
		},
		{
			name: "in.SetFields",
			f: func(ctx context.Context, v any) context.Context {
				return in.SetFields(ctx, map[string]any{key: v})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := redactable("hello")

			ctx := context.Background()
			ctx = tt.f(ctx, r)

			v, ok := ctxerr.Fields(ctx)[key]
			if !ok {
				t.Fatal("key not set")
			}

			if v != "redacted" {
				t.Error("hook did not work", v)
			}
		})
	}
}

func TestGlobalFieldsHook(t *testing.T) {
	ctxerr.AddFieldHook(func(_ context.Context, a any) any { return a })
}

type FieldError struct {
	fields map[string]any
	err    error
}

func (fe FieldError) Error() string {
	return fe.err.Error()
}

func (fe FieldError) Fields() map[string]any {
	return fe.fields
}

func (fe FieldError) Unwrap() error {
	return errors.Unwrap(fe.err)
}

func NewFieldError(msg string, fields map[string]any) error {
	return FieldError{
		fields: fields,
		err:    fmt.Errorf(msg),
	}
}

func WrapFieldError(err error, msg string, fields map[string]any) error {
	return FieldError{
		fields: fields,
		err:    fmt.Errorf("%s : %w", msg, err),
	}
}

func TestAllFieldsWithMultipleTypesOfErrors(t *testing.T) {
	var _ error = FieldError{}

	err := NewFieldError("bottom", map[string]any{"a": "a"})
	err = fmt.Errorf("fmt : %w", err)
	ctx := ctxerr.SetField(context.Background(), "b", "b")
	ctx = ctxerr.SetCategory(ctx, "category1")
	err = ctxerr.Wrap(ctx, err, "CTXERR_CODE_1", "ctxerr1")
	ctx = ctxerr.SetCategory(ctx, "category2")
	err = ctxerr.Wrap(ctx, err, "CTXERR_CODE_2", "ctxerr2")
	err = WrapFieldError(err, "wrapfe", map[string]any{"c": "c"})

	expectedMessage := "wrapfe : ctxerr2 : ctxerr1 : fmt : bottom"
	if msg := err.Error(); msg != expectedMessage {
		t.Errorf("message didn't match \n'%s'\n'%s'", msg, expectedMessage)
	}

	f := ctxerr.AllFields(err)
	expectedFields := map[string]any{
		"a":              "a",
		"b":              "b",
		"c":              "c",
		"error_code":     "CTXERR_CODE_1",
		"error_category": "category1",
		"error_location": []any{
			"ctxerr_test.TestAllFieldsWithMultipleTypesOfErrors",
			"ctxerr_test.TestAllFieldsWithMultipleTypesOfErrors",
		},
	}
	if !reflect.DeepEqual(f, expectedFields) {
		t.Errorf("fields didn't match \n%#v\n%#v", f, expectedFields)
	}

	if !ctxerr.HasField(err, "a") {
		t.Error("missing in HasField")
	}

	if !ctxerr.HasCategory(err, "category2") {
		t.Error("missing category2")
	}
	if !ctxerr.HasCategory(err, "category1") {
		t.Error("missing category1")
	}
}

type OtherFieldFuncError struct {
	fields map[string]any
	err    error
}

func (offe OtherFieldFuncError) Error() string {
	return offe.err.Error()
}

func (offe OtherFieldFuncError) FieldsMap() map[string]any {
	return offe.fields
}

func (offe OtherFieldFuncError) Unwrap() error {
	return errors.Unwrap(offe.err)
}

type IFieldsMap interface {
	FieldsMap() map[string]any
}

func NewOtherFieldFuncError(msg string, fields map[string]any) error {
	return OtherFieldFuncError{
		fields: fields,
		err:    fmt.Errorf(msg),
	}
}

func TestAddFieldsFuncs(t *testing.T) {
	in := ctxerr.NewInstance()
	in.AddFieldsFunc(func(err error) map[string]any {
		if v, ok := err.(IFieldsMap); ok {
			return v.FieldsMap()
		}
		return nil
	})

	err := NewOtherFieldFuncError("msg", map[string]any{"a": "a"})
	if in.AllFields(err)["a"] != "a" {
		t.Error("field not set from interface")
	}

	err = ctxerr.New(ctxerr.SetField(context.Background(), "b", "b"), "CODE", "msg")
	if in.AllFields(err)["b"] != "b" {
		t.Error("field not set from default interface")
	}

	// Ensure empty funcs defaults
	in.GetFieldsFuncs = nil
	ctx := ctxerr.SetField(context.Background(), "c", "c")
	ctx = ctxerr.SetCategory(ctx, "foo")
	err = ctxerr.New(ctx, "CODE", "msg")
	if in.AllFields(err)["c"] != "c" {
		t.Error("field not set from default interface fallback")
	}
	if !in.HasCategory(err, "foo") {
		t.Error("missing category")
	}
	if !in.HasField(err, "c") {
		t.Error("missing hasField")
	}
}

func TestGlobaTestAddFieldsFuncs(t *testing.T) {
	ctxerr.AddFieldsFunc(func(_ error) map[string]any { return nil })
}

func TestJoined(t *testing.T) {
	actx := ctxerr.SetField(context.Background(), "a", "a")
	actx = ctxerr.SetCategory(actx, "cat_a")
	a := ctxerr.New(actx, "CODE_A", "msg_a")

	bctx := ctxerr.SetField(context.Background(), "b", "b")
	bctx = ctxerr.SetCategory(bctx, "cat_b")
	b := ctxerr.New(bctx, "CODE_B", "msg_b")

	cctx := ctxerr.SetField(context.Background(), "c", "c")
	c := ctxerr.Wrap(cctx, errors.Join(a, b), "CODE_C", "msg_c")

	if !ctxerr.HasCategory(c, "cat_a") {
		t.Error("missing category cat_a")
	}
	if !ctxerr.HasCategory(c, "cat_b") {
		t.Error("missing category cat_b")
	}

	if !ctxerr.HasField(c, "a") {
		t.Error("missing field a")
	}
	if !ctxerr.HasField(c, "b") {
		t.Error("missing field b")
	}

	f := ctxerr.AllFields(c)
	expectedFields := map[string]any{
		"a":              "a",
		"b":              "b",
		"c":              "c",
		"error_code":     "CODE_B",
		"error_category": "cat_b",
		"error_location": []any{
			"ctxerr_test.TestJoined",
			"ctxerr_test.TestJoined",
			"ctxerr_test.TestJoined",
		},
	}

	if !reflect.DeepEqual(f, expectedFields) {
		t.Errorf("fields didn't match \n%#v\n%#v", f, expectedFields)
	}
}
