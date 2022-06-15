/*
Package ctxerr is a way of creating and handling errors with extra context.

Note: Errors can be wrapped as many times as wanted but should only be handled once!


New(f) and Wrap(f)

Creating a new error or wrapping an error are as simple as:
	ctxerr.New(ctx, "<code>", "<message>")
	ctxerr.Newf(ctx, "<code>", "%s", "<message>")
	ctxerr.Wrap(ctx, err, "<code>", "<message>")

A quick wrap function is available to avoid needing to create unused codes and messages.
This function calls Wrap with an empty string for the code and calls "ctxerr.CallerFunc(1)" for the message.
	ctxerr.QuickWrap(ctx, err)

Note: Wrapping nil will return nil.


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


Handle

Handle exists to make sure all errors are handled in the say way.
It should be called only once at the top of all wrapped errors.
It will run through all hooks added through configuration or fallback to the DefaultLogHook.


(i.e. HTTP handle functions or goroutines)
	nctx := ctxerr.SetFields(context.Background(), ctxerr.Fields(ctx))
	go func(ctx context.Context){
		if err := foo(ctx) {
			err = ctxerr.QuickWrap(ctx, err)
			ctxerr.Handle(err)
		}
	}(nctx)


Configuration

Hooks can be used to edit the context before creating the error and to handle the error.

If you need the context to change prior to creation of the error use 'AddCreateHook'.
	ctxerr.AddCreateHook(customHook)

To change how errors are handled use 'AddHandleHook'.
Note: If you are not adding a custom logging hook it may be useful to add the default.
	ctxerr.AddHandleHook(metricOnError)
	ctxerr.AddHandleHook(DefaultLogHook)


HTTP

There is an http subpackage for handling HTTP errors.
The function included returns a standardized struct filled in with details of the error.
There are fields key constansts to help with this.
	ctx = ctxerr.SetHTTPStatusCode(ctx, http.StatusBadRequest)
	ctx = ctxerr.SetAction(ctx, "action for a user to understand how to fix the error if they can")
*/
package ctxerr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
)

func init() {
	// Always add the code to the fields
	AddCreateHook(setCodeHook)
}

var createHooks []func(ctx context.Context, code string, wrapping error) context.Context
var handleHooks []func(error)

const (
	// FieldKeyCode should be unique to the error
	FieldKeyCode = "error_code"
	// FieldKeyStatusCode can be used to choose a status code to return to http request
	FieldKeyStatusCode = "error_status_code"
	// FieldKeyAction is an action that a user can take to fix the error
	FieldKeyAction = "error_action"
	//FieldKeyCategory can be used with IsCategorgy(...) to determin a category of error
	FieldKeyCategory = "error_category"
)

// FieldsKey is the key used to add and decode fields on the context
// Change or use it in other packages if you want to unify fields
var FieldsKey interface{} = contextKey("fields")

// Handle should be called one per error to handle it when it can no logger be returned
func Handle(err error) {
	if err == nil {
		return
	}

	if len(handleHooks) == 0 {
		DefaultLogHook(err)
		return
	}

	for _, hook := range handleHooks {
		hook(err)
	}
}

// AddCreateHook adds a hooks that is called to update the context before the error is created
func AddCreateHook(f func(ctx context.Context, code string, wrapping error) context.Context) {
	createHooks = append(createHooks, f)
}

// AddHandleHook adds a hook to be run on handling of an error
func AddHandleHook(f func(error)) {
	handleHooks = append(handleHooks, f)
}

// CtxErr is the interface that should be checked in a errors.As function
type CtxErr interface {
	error
	Unwrap() error
	Is(error) bool
	As(interface{}) bool

	Fields() map[string]interface{}
	Context() context.Context
	WithContext(context.Context)
}

// New creates a new error
func New(ctx context.Context, code string, message ...interface{}) error {
	for _, hook := range createHooks {
		ctx = hook(ctx, code, nil)
	}

	return &impl{
		ctx: ctx,
		msg: fmt.Sprint(message...),
	}
}

// Newf creates a new error message formatting
func Newf(ctx context.Context, code, message string, messageArgs ...interface{}) error {
	for _, hook := range createHooks {
		ctx = hook(ctx, code, nil)
	}

	return &impl{
		ctx: ctx,
		msg: fmt.Sprintf(message, messageArgs...),
	}
}

// Wrap creates a new error with another wrapped under it
func Wrap(ctx context.Context, err error, code string, message ...interface{}) error {
	if err == nil {
		return nil
	}

	for _, hook := range createHooks {
		ctx = hook(ctx, code, err)
	}

	return &impl{
		ctx:     ctx,
		msg:     fmt.Sprint(message...),
		wrapped: err,
	}
}

// Wrapf creates a new error with a formatted message with another wrapped under it
func Wrapf(ctx context.Context, err error, code, message string, messageArgs ...interface{}) error {
	if err == nil {
		return nil
	}

	for _, hook := range createHooks {
		ctx = hook(ctx, code, err)
	}

	return &impl{
		ctx:     ctx,
		msg:     fmt.Sprintf(message, messageArgs...),
		wrapped: err,
	}
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
	f := map[string]interface{}{}
	for k, v := range Fields(ctx) {
		f[k] = v
	}
	f[key] = value
	return context.WithValue(ctx, FieldsKey, f)
}

// SetFields can add multiple fields onto the context
func SetFields(ctx context.Context, fields map[string]interface{}) context.Context {
	f := map[string]interface{}{}
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

// AllFields unwraps the error collecting/replacing fields as it goes down the tree
func AllFields(err error) map[string]interface{} {
	f := map[string]interface{}{}
	var e CtxErr = &impl{}
	for {
		if err == nil {
			return f
		}
		if ok := errors.As(err, &e); !ok {
			return f
		}
		for k, v := range e.Fields() {
			f[k] = v
		}
		err = errors.Unwrap(err)
	}
}

// As is a shorthand for errors.As and includes an ok
func As(err error) (CtxErr, bool) {
	if err == nil {
		return nil, false
	}

	var e CtxErr = &impl{}
	if ok := errors.As(err, &e); !ok {
		return nil, false
	}

	return e, true
}

// HasCategory tells if an error in the chain matches the category
func HasCategory(err error, category interface{}) bool {
	if err == nil {
		return false
	}
	var e CtxErr = &impl{}
	if !errors.As(err, &e) {
		return false
	}
	for {
		if c, ok := e.Fields()[FieldKeyCategory]; ok {
			if c == category {
				return true
			}
		}
		u := errors.Unwrap(e)
		if errors.As(u, &e) {
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
	if u := errors.Unwrap(im); u != nil {
		return im.msg + " : " + u.Error()
	}
	return im.msg
}

// Unwrap fulfills the interface to allow errors.Unwrap
func (im *impl) Unwrap() error { return im.wrapped }

// As Fulfills the As interface to know if something is the same type
func (im *impl) As(err interface{}) bool {
	_, ok := err.(CtxErr)
	return ok
}

// Is fulfills the interface to allow errors.Is
func (im *impl) Is(err error) bool { return im.As(err) }

// Context retrieves the context passed in when the error was created
func (im *impl) Context() context.Context { return im.ctx }

// Fields retrieves the fields from the context passed in when the error was created
func (im *impl) Fields() map[string]interface{} { return Fields(im.ctx) }

// WithContext replaces the context of the error
func (im *impl) WithContext(ctx context.Context) { im.ctx = ctx }

// ** Helper Functions ** //

// SetHTTPStatusCode is equivelent to ctxerr.SetField(ctx, FieldKeyStatusCode, code)
func SetHTTPStatusCode(ctx context.Context, code int) context.Context {
	return SetField(ctx, FieldKeyStatusCode, code)
}

// SetAction is equivelent to ctxerr.SetField(ctx, FieldKeyAction, action)
func SetAction(ctx context.Context, action string) context.Context {
	return SetField(ctx, FieldKeyAction, action)
}

// SetCategory is equivelent to ctxerr.SetField(ctx, FieldKeyStatusCode, category)
func SetCategory(ctx context.Context, category interface{}) context.Context {
	return SetField(ctx, FieldKeyCategory, category)
}

// ** Hooks ** //

// DefaultLogHook is the default hook used log errors
// It is the fallback if there are no other handle hooks
func DefaultLogHook(err error) {
	f := AllFields(err)
	b, merr := json.Marshal(f)
	fields := string(b)
	if merr != nil {
		fields = fmt.Sprintf("fields '%v' could not be marshalled as JSON: %s", f, merr)
	}
	log.Printf("%s - %s", err, fields)
}

func setCodeHook(ctx context.Context, code string, wrapping error) context.Context {
	if code != "" {
		ctx = SetField(ctx, FieldKeyCode, code)
	}
	return ctx
}
