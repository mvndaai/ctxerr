/*
Package ctxerr is a way of creating and handling errors with extra context.

Note: Errors can be wrapped as many times as wanted but should only be handled once!


New and Wrap

Creating a new error or wrapping an error are as simple as:
	ctxerr.New(ctx, "<code>", "<message>")
	ctxerr.Wrap(ctx, err, "<code>", "<message>")

A quick wrap function is available to avoid needing to create unused codes and messages.
This function calls Wrap with an empty string for the code and calls "ctxerr.CallerFunc(1)" for the message.
	ctxerr.QuickWrap(ctx, err)

Note: Wrapping nil will return nil. This enables returning wraps extra error checking.

Note: If an empty string is passed as the code, the OnEmptyCode function will be called.
The default behavior is to log a warning.


Context

A context is passed in so that anywhere in code more information can be added.
Adding information (aka fields) to a context is done by:
	ctx = ctxerr.SetField(ctx, "field", "value")
	ctx = ctxerr.SetFields(ctx, map[string]interface{}{"foo": "bar", "baz": 0})

Some common field keys have been predefined to be used in this or sub packages.
This includes 'FieldKeyCode' which is used to set the 'code' passed into the New/Wrap functions on the context.
See the HTTP section below for more examples.

The function 'Fields' allows retrieving the fields added to the context.
Using this for goroutines ensures all the data gets propagated.
	nctx := ctxerr.SetFields(context.Background(), ctxerr.Fields(ctx))
	go foo(nctx)


Logging

Log functions with different severities are provided to ensure all errors all logged the same way.
By default, the functions call 'log.Println' with the fields as a marshaled JSON object in addition to the message chain.
The functions are public vars so they can be replaced for any logging setup.
	ctxerr.LogDebug = func(err error) { customLogFunc(err) }
	ctxerr.LogDebug(err)
	ctxerr.LogError(err)


Configurable Functions

The configurable functions (OnNew, Handle, and OnEmptyCode) are public vars so they can be customized.

OnNew is automatically called on a New or Wrap. Replace it to add custom fields on creation.
The default is to add the code as a field on the context.

OnEmptyCode is automatically called when creating an error that has a code of an empty string.
The default behavior is to unwrap the error looking for a previous code.
If one does not exist an error will be logged.

OnHandle exists to make sure all errors are handled in the say way.
The default behavior is to log the error.
It should be called only once at the top of all wrapped errors.
(i.e. HTTP handle functions or goroutines)
	nctx := ctxerr.SetFields(context.Background(), ctxerr.Fields(ctx))
	go func(ctx context.Context){
		if err := foo(ctx) {
			err = ctxerr.QuickWrap(ctx, err)
			ctxerr.OnHandle(err)
		}
	}(nctx)

HTTP

There is an http subpackage for handling HTTP errors.
The function included returns a standardized struct filled in with details of the error.
There are fields key constansts to help with this.
	ctx = ctxerr.SetField(ctx, ctxerr.FieldStatusCode, 400)
	ctx = ctxerr.SetField(ctx, ctxerr.FieldKeyAction, "action for an outside user")
*/
package ctxerr

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/xerrors"
)

const (
	// FieldKeyCode should be unique to the error
	FieldKeyCode = "error_code"
	// FieldKeyStatusCode can be used to choose a status code to return to http request
	FieldKeyStatusCode = "error_status_code"
	// FieldKeyAction is an action that a user can take to fix the error
	FieldKeyAction = "error_action"
	//FieldKeyCategory can be used with IsCategorgy(...) to determin a category of error
	FieldKeyCategory = "error_category"
	// FieldKeyRelatedCode can be used when there is a second error but want the original code in the logs
	FieldKeyRelatedCode = "error_related_code"

	// fieldKeyLogSeverity is the log severity
	fieldKeyLogSeverity = "error_severity"
	//fieldKeyFunctionName is the name a of function added to logging of Quickwraps
	fieldKeyFunctionName = "error_function_name"
)

// Customizations for the package
var (
	// OnNew is called when an error is created by New, Wrap, or QuickWrap
	OnNew = DefaultOnNew
	// OnEmptyCode is called when an error is created with an empty string for a code
	OnEmptyCode = DefaultOnEmptyCode
	// OnHandle should be called when handling an error
	Handle = DefaultHandler

	//FieldsKey is the key used to add and decode fields on the context
	FieldsKey interface{} = contextKey("fields")
)

// Configurable logging methods
var (
	LogDebug = DefaultLog("debug")
	LogInfo  = DefaultLog("info")
	LogWarn  = DefaultLog("warn")
	LogError = DefaultLog("error")
	LogFatal = DefaultLog("fatal")
)

// CtxErr is the interface that should be checked in a xerrors As function
// TODO revisit this comment
type CtxErr interface {
	error
	xerrors.Wrapper
	Is(error) bool
	As(interface{}) bool

	Fields() map[string]interface{}
	Context() context.Context
	WithContext(context.Context)
}

// New creates a new error
func New(ctx context.Context, code, message string, messageArgs ...interface{}) error {
	var e CtxErr = &impl{
		ctx: OnNew(ctx, code),
		msg: fmt.Sprintf(message, messageArgs...),
	}
	if code == "" {
		return OnEmptyCode(e)
	}
	return e
}

// Wrap creates a new error with another wrapped under it
func Wrap(ctx context.Context, err error, code, message string, messageArgs ...interface{}) error {
	if err == nil {
		return nil
	}
	var e CtxErr = &impl{
		ctx:     OnNew(ctx, code),
		msg:     fmt.Sprintf(message, messageArgs...),
		wrapped: err,
	}
	if code == "" {
		return OnEmptyCode(e)
	}
	return e
}

// QuickWrap will wrap an error with an empty code and the calling function's name as the message
func QuickWrap(ctx context.Context, err error) error {
	return Wrap(ctx, err, "", CallerFunc(1))
}

// Fields retrieves the fields from the context
func Fields(ctx context.Context) map[string]interface{} {
	fi := ctx.Value(FieldsKey)
	if fi == nil {
		return nil
	}
	if f, ok := fi.(map[string]interface{}); ok {
		return f
	}
	return nil
}

// SetField adds a field onto the context
func SetField(ctx context.Context, key string, value interface{}) context.Context {
	f := make(map[string]interface{}, 0)
	for k, v := range Fields(ctx) {
		f[k] = v
	}
	f[key] = value
	return context.WithValue(ctx, FieldsKey, f)
}

// SetFields can add multiple fields onto the context
func SetFields(ctx context.Context, fields map[string]interface{}) context.Context {
	f := make(map[string]interface{}, 0)
	for k, v := range Fields(ctx) {
		f[k] = v
	}
	for k, v := range fields {
		f[k] = v
	}
	return context.WithValue(ctx, FieldsKey, f)
}

//CallerFunc gets the name of the calling function
func CallerFunc(skip int) string {
	if pc, _, _, ok := runtime.Caller(skip + 1); ok {
		if details := runtime.FuncForPC(pc); details != nil {
			return filepath.Base(details.Name())
		}
	}
	return "caller location unretrievable"
}

// DefaultOnNew is the default OnNew function called when creating errors
func DefaultOnNew(ctx context.Context, code string) context.Context {
	if code != "" {
		ctx = SetField(ctx, FieldKeyCode, code)
	}
	return ctx
}

// DefaultOnEmptyCode is the default way to handle errors what have an empty string as their code
func DefaultOnEmptyCode(err CtxErr) CtxErr {
	var current error = err
	var i CtxErr = &impl{}
	for {
		if ok := xerrors.As(current, &i); !ok {
			break
		}
		f := i.Fields()
		if f != nil {
			if _, ok := f[FieldKeyCode]; ok {
				return err
			}
		}
		current = xerrors.Unwrap(current)
	}

	LogWarn(&impl{
		ctx:     context.Background(),
		msg:     "did not find a code within the error's stack",
		wrapped: err,
	})

	err.WithContext(SetField(err.Context(), FieldKeyCode, "no_code"))
	return err
}

// DefaultHandler is the default way of handling an error, logging with the fields
func DefaultHandler(err error) {
	if err == nil {
		return
	}
	logger := LogError
	if de := Deepest(err); de != nil {
		if v, ok := de.Fields()[FieldKeyStatusCode]; ok {
			if strings.HasPrefix(fmt.Sprint(v), "4") {
				logger = LogWarn
			}
		}
	}
	logger(err)
}

// Deepest retrieves the deepest CtxErr or nil if one does not exist
func Deepest(err error) CtxErr {
	if err == nil {
		return nil
	}

	var e CtxErr = &impl{}
	if ok := xerrors.Is(err, e); !ok {
		return nil
	}
	xerrors.As(err, &e)

	for {
		u := xerrors.Unwrap(e)
		if ok := xerrors.Is(u, e); ok {
			xerrors.As(u, &e)
			continue
		}
		break
	}
	return e
}

// IsCategory tells if an error in the chain matches the category
func IsCategory(err error, category interface{}) bool {
	if err == nil {
		return false
	}
	var e CtxErr = &impl{}
	if ok := xerrors.Is(err, e); !ok {
		return false
	}
	xerrors.As(err, &e)
	for {
		if c, ok := e.Fields()[FieldKeyCategory]; ok {
			if c == category {
				return true
			}
		}
		u := xerrors.Unwrap(e)
		if ok := xerrors.Is(u, e); ok {
			xerrors.As(u, &e)
			continue
		}
		break
	}
	return false
}

/* Implementation helper code */

type contextKey string

type impl struct {
	ctx     context.Context
	msg     string
	wrapped error
}

// Error fulfills the error interface
func (im *impl) Error() string {
	if u := xerrors.Unwrap(im); u != nil {
		return im.msg + " : " + u.Error()
	}
	return im.msg
}

// Unwrap fulfills the xerrors Wrapper interface
func (im *impl) Unwrap() error { return im.wrapped }

// As Fulfills the As interface to know if something is the same type
func (im *impl) As(err interface{}) bool {
	_, ok := err.(CtxErr)
	return ok
}

// Is fulfills the interface to allow xerrors.Is
func (im *impl) Is(err error) bool { return im.As(err) }

// Context retrieves the context passed in when the error was created
func (im *impl) Context() context.Context { return im.ctx }

// Fields retrieves the fields from the context passed in when the error was created
func (im *impl) Fields() map[string]interface{} { return Fields(im.ctx) }

// WithContext replaces the context of the error
func (im *impl) WithContext(ctx context.Context) { im.ctx = ctx }

// DefaultLog is the default logging function
func DefaultLog(severity string) func(error) {
	return func(err error) {
		f := make(map[string]interface{}, 0)
		if e := Deepest(err); e != nil {
			f = e.Fields()
		}
		f[fieldKeyLogSeverity] = severity

		b, merr := json.Marshal(f)
		fields := string(b)
		if merr != nil {
			fields = fmt.Sprintf("fields '%v' could not be marshalled as JSON: %s", f, merr)
		}
		log.Printf("%s - %s", err, fields)
	}
}
