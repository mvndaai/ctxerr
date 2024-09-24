/*
Package ctxerr is a way of creating and handling errors with extra context.

Note: Errors can be wrapped as many times as wanted but should only be handled once!

New(f) and Wrap(f)

Creating a new error or wrapping an error are as simple as:

	ctxerr.New(ctx, "<code>", "<message>")
	ctxerr.Newf(ctx, "<code>", "%s", "<vars>")
	ctxerr.Wrap(ctx, err, "<code>", "<message>")
	ctxerr.Wrapf(ctx, err, "<code>", "%s", "<var>")

A quick wrap function is available to avoid needing to create unused codes and messages.
This function calls Wrap with an empty string for the code no message.

	ctxerr.QuickWrap(ctx, err)

Note: Wrapping nil will return nil.

# Context

A context is passed in so that anywhere in code more information can be added.
Adding information (aka fields) to a context is done by:

	ctx = ctxerr.SetField(ctx, "field", "value")
	ctx = ctxerr.SetFields(ctx, map[string]any{"foo": "bar", "baz": 0})

Some common field keys have been predefined to be used in this or sub packages.
This includes 'FieldKeyCode' which is used to set the 'code' passed into the New/Wrap functions on the context.
See the HTTP section below for more examples.

The function 'Fields' allows retrieving the fields added to the context.
Using this for goroutines ensures all the data gets propagated.

	nctx := ctxerr.SetFields(context.Background(), ctxerr.Fields(ctx))
	go foo(nctx)

# Handle

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

# Configuration

Hooks can be used to edit the context before creating the error and to handle the error.

If you need the context to change prior to creation of the error use 'AddCreateHook'.

	ctxerr.AddCreateHook(customHook)

To change how errors are handled use 'AddHandleHook'.
Note: If you are not adding a custom logging hook it may be useful to add the default.

	ctxerr.AddHandleHook(metricOnError)
	ctxerr.AddHandleHook(DefaultLogHook)

There is an http subpackage for handling HTTP errors.
The function included returns a standardized struct filled in with details of the error.
There are fields key constansts to help with this.

	ctx = ctxerr.SetHTTPStatusCode(ctx, http.StatusBadRequest)
	ctx = ctxerr.SetAction(ctx, "action for a user to understand how to fix the error if they can")

An "Action" is a user facing error that a user can take an action on to fix.
There are helper http functions that set the status code and action in one call.

	ctxerr.NewHTTP(ctx, "<code>", "<action>", http.StatusBadRequest, "<message>")
	ctxerr.NewHTTPf(ctx, "<code>", "<action>", http.StatusConflict, "%s", "<vars>")
	ctxerr.WrapHTTP(ctx, err, "<code>", "<action>", http.StatusBadRequest, "<message>")
	ctxerr.WrapHTTPf(ctx, err, "<code>", "<action>", http.StatusBadRequest, "%s", "<vars>")
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
	"strings"

	"github.com/mvndaai/ctxerr/joinederr"
)

var global Instance

func init() {
	global = NewInstance()
}

// Instance creates a local instance so you can have a different setup than global
type Instance struct {
	// CreateHooks are functions that run on creation to set fields on context
	CreateHooks []func(ctx context.Context, code string, wrapping error) context.Context
	// HandleHooks are functions that run on ctxerr.Handle
	HandleHooks []func(error)
	// FieldHooks are functions that run on ctxerr.SetField(s)
	FieldHooks []func(context.Context, any) any
	// FieldsAsSlice are keys that get gathered as a slice in ctxerr.AllFields
	FieldsAsSlice []string
	// GetFieldsFuncs are functions that get the fieldss from an error
	GetFieldsFuncs []func(error) map[string]any
}

// NewInstance creates a local instance with the default create hooks
func NewInstance() Instance {
	in := Instance{}
	// Always add the code to the fields
	in.AddCreateHook(SetCodeHook)
	// Always add the location of where the error happened
	in.AddCreateHook(SetLocationHook)
	// Gather keys like location as slice instead of just the deepest value
	in.FieldsAsSlice = []string{FieldKeyLocation}
	// No built in hooks
	in.FieldHooks = []func(context.Context, any) any{}
	// Functions for getting the fields
	in.GetFieldsFuncs = append(in.GetFieldsFuncs, DefaultFieldsFunc)
	return in
}

const (
	// FieldKeyCode should be unique to the error
	FieldKeyCode = "error_code"
	// FieldKeyStatusCode can be used to choose a status code to return to http request
	FieldKeyStatusCode = "error_status_code"
	// FieldKeyAction is an action that a user can take to fix the error
	FieldKeyAction = "error_action"
	// FieldKeyCategory can be used with IsCategorgy(...) to determin a category of error
	FieldKeyCategory = "error_category"
	// FieldKeyLocation shows the file location of the err
	FieldKeyLocation = "error_location"
)

// FieldsKey is the key used to add and decode fields on the context
// Change or use it in other packages if you want to unify fields
var FieldsKey any = contextKey("fields")

// Handle should be called one per error to handle it when it can no logger be returned
func Handle(err error) { global.Handle(err) }
func (in Instance) Handle(err error) {
	if err == nil {
		return
	}

	if len(in.HandleHooks) == 0 {
		in.DefaultLogHook(err)
		return
	}

	for _, hook := range in.HandleHooks {
		hook(err)
	}
}

// AddCreateHook adds a hooks that is called to update the context before the error is created
func AddCreateHook(f func(ctx context.Context, code string, wrapping error) context.Context) {
	global.AddCreateHook(f)
}
func (in *Instance) AddCreateHook(f func(ctx context.Context, code string, wrapping error) context.Context) {
	if in == nil {
		// cannot return an error so adding info to panic
		panic("cannot call AddCreateHook because ctxerr.Instance is nil")
	}
	in.CreateHooks = append(in.CreateHooks, f)
}

// AddHandleHook adds a hook to be run on handling of an error
func AddHandleHook(f func(error)) { global.AddHandleHook(f) }
func (in *Instance) AddHandleHook(f func(error)) {
	if in == nil {
		// cannot return an error so adding info to panic
		panic("cannot call AddHandleHook because ctxerr.Instance is nil")
	}
	in.HandleHooks = append(in.HandleHooks, f)
}

// AddFieldHooks adds a hook to be run on handling of an error
func AddFieldHook(f func(context.Context, any) any) { global.AddFieldHook(f) }
func (in *Instance) AddFieldHook(f func(context.Context, any) any) {
	if in == nil {
		// cannot return an error so adding info to panic
		panic("cannot call AddFieldHooks because ctxerr.Instance is nil")
	}
	in.FieldHooks = append(in.FieldHooks, f)
}

// AddFieldsFuncs adds a function that can be used to get fields from an error
func AddFieldsFunc(f func(error) map[string]any) { global.AddFieldsFunc(f) }
func (in *Instance) AddFieldsFunc(f func(error) map[string]any) {
	if in == nil {
		// cannot return an error so adding info to panic
		panic("cannot call AddFieldsFuncs because ctxerr.Instance is nil")
	}
	in.GetFieldsFuncs = append(in.GetFieldsFuncs, f)
}

// CtxErr is the interface that should be checked in a errors.As function
type CtxErr interface {
	error
	Unwrap() error
	Is(error) bool
	As(any) bool

	Fields() map[string]any
	Context() context.Context
	WithContext(context.Context)
}

// New creates a new error
func New(ctx context.Context, code string, message ...any) error {
	return global.New(ctx, code, message...)
}
func (in Instance) New(ctx context.Context, code string, message ...any) error {
	for _, hook := range in.CreateHooks {
		ctx = hook(ctx, code, nil)
	}

	im := &impl{ctx: ctx}
	if len(message) > 0 && message[0] != nil {
		im.msg = fmt.Sprint(message...)
	}
	return im
}

// Newf creates a new error message formatting
func Newf(ctx context.Context, code, message string, messageArgs ...any) error {
	return global.Newf(ctx, code, message, messageArgs...)
}
func (in Instance) Newf(ctx context.Context, code, message string, messageArgs ...any) error {
	for _, hook := range in.CreateHooks {
		ctx = hook(ctx, code, nil)
	}

	return &impl{
		ctx: ctx,
		msg: fmt.Sprintf(message, messageArgs...),
	}
}

// Wrap creates a new error with another wrapped under it
func Wrap(ctx context.Context, err error, code string, message ...any) error {
	return global.Wrap(ctx, err, code, message...)
}

func (in Instance) Wrap(ctx context.Context, err error, code string, message ...any) error {
	if err == nil {
		return nil
	}

	for _, hook := range in.CreateHooks {
		ctx = hook(ctx, code, err)
	}

	im := &impl{
		ctx:     ctx,
		wrapped: err,
	}

	if len(message) > 0 && message[0] != nil {
		im.msg = fmt.Sprint(message...)
	}
	return im
}

// Wrapf creates a new error with a formatted message with another wrapped under it
func Wrapf(ctx context.Context, err error, code, message string, messageArgs ...any) error {
	return global.Wrapf(ctx, err, code, message, messageArgs...)
}
func (in Instance) Wrapf(ctx context.Context, err error, code, message string, messageArgs ...any) error {
	if err == nil {
		return nil
	}

	for _, hook := range in.CreateHooks {
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
	return global.Wrap(ctx, err, "", nil)
}
func (in Instance) QuickWrap(ctx context.Context, err error) error {
	return in.Wrap(ctx, err, "", nil)
}

// Fields retrieves the fields from the context
func Fields(ctx context.Context) map[string]any {
	if ctx == nil {
		return nil
	}
	fi := ctx.Value(FieldsKey)
	if fi == nil {
		return nil
	}
	if f, ok := fi.(map[string]any); ok {
		return f
	}
	return nil
}

// SetField adds a field onto the context
func SetField(ctx context.Context, key string, value any) context.Context {
	return global.SetField(ctx, key, value)
}
func (in Instance) SetField(ctx context.Context, key string, value any) context.Context {
	for _, f := range in.FieldHooks {
		value = f(ctx, value)
	}
	f := map[string]any{}
	for k, v := range Fields(ctx) {
		f[k] = v
	}
	f[key] = value
	return context.WithValue(ctx, FieldsKey, f)
}

// SetFields can add multiple fields onto the context
func SetFields(ctx context.Context, fields map[string]any) context.Context {
	return global.SetFields(ctx, fields)
}
func (in Instance) SetFields(ctx context.Context, fields map[string]any) context.Context {
	f := map[string]any{}
	for k, v := range Fields(ctx) {
		f[k] = v
	}
	for k, v := range fields {
		for _, f := range in.FieldHooks {
			v = f(ctx, v)
		}
		f[k] = v
	}
	return context.WithValue(ctx, FieldsKey, f)
}

// CallerFunc gets the name of the calling function
func CallerFunc(skip int) string {
	f := "caller location unretrievable"
	if pc, _, _, ok := runtime.Caller(skip + 1); ok {
		if details := runtime.FuncForPC(pc); details != nil {
			f = filepath.Base(details.Name())
		}
	}

	// For helper functions QuickWrap and we still want the hook getting the location
	if strings.HasPrefix(f, "ctxerr.") {
		return CallerFunc(skip + 2)
	}

	return f
}

// CallerFuncs is a shortcut for calling CallerFunc many times
func CallerFuncs(skip, depth int) []string {
	f := []string{}
	for i := 1; i < depth+1; i++ {
		f = append(f, CallerFunc(skip+i))
	}
	return f
}

// AllFields unwraps the error collecting/replacing fields as it goes down the tree
func AllFields(err error) map[string]any { return global.AllFields(err) }
func (in Instance) AllFields(err error) map[string]any {
	f := map[string]any{}
	fieldFuncs := append([]func(error) map[string]any{}, in.GetFieldsFuncs...)
	if len(fieldFuncs) == 0 {
		fieldFuncs = append(fieldFuncs, DefaultFieldsFunc)
	}

	iter := joinederr.NewDepthFirstIterator(err)
	for {
		err = iter.Next()
		if err == nil {
			return f
		}

		fields := map[string]any{}
		for _, fn := range fieldFuncs {
			for k, v := range fn(err) {
				fields[k] = v
			}
		}
	OUTER:
		for k, v := range fields {
			for _, sk := range in.FieldsAsSlice {
				if k == sk {
					if _, ok := f[k]; !ok {
						f[k] = []any{}
					}

					f[k] = append(f[k].([]any), v)
					continue OUTER
				}
			}
			f[k] = v
		}
	}
}

// HasField unwraps and checks if the error has a field in the error tree
func HasField(err error, field string) bool { return global.HasField(err, field) }
func (in Instance) HasField(err error, field string) bool {
	fieldFuncs := append([]func(error) map[string]any{}, in.GetFieldsFuncs...)
	if len(fieldFuncs) == 0 {
		fieldFuncs = append(fieldFuncs, DefaultFieldsFunc)
	}

	iter := joinederr.NewDepthFirstIterator(err)
	for {
		err = iter.Next()
		if err == nil {
			return false
		}

		fields := map[string]any{}
		for _, fn := range fieldFuncs {
			for k, v := range fn(err) {
				fields[k] = v
			}
		}

		if _, ok := fields[field]; ok {
			return true
		}
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
func HasCategory(err error, category any) bool { return global.HasCategory(err, category) }
func (in Instance) HasCategory(err error, category any) bool {
	fieldFuncs := append([]func(error) map[string]any{}, in.GetFieldsFuncs...)
	if len(fieldFuncs) == 0 {
		fieldFuncs = append(fieldFuncs, DefaultFieldsFunc)
	}

	iter := joinederr.NewDepthFirstIterator(err)
	for {
		err = iter.Next()
		if err == nil {
			return false
		}

		fields := map[string]any{}
		for _, fn := range fieldFuncs {
			for k, v := range fn(err) {
				fields[k] = v
			}
		}

		if c, ok := fields[FieldKeyCategory]; ok {
			if c == category {
				return true
			}
		}
	}
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
		if im.msg == "" {
			return u.Error()
		}
		return im.msg + " : " + u.Error()
	}
	return im.msg
}

// Unwrap fulfills the interface to allow errors.Unwrap
func (im *impl) Unwrap() error { return im.wrapped }

// As Fulfills the As interface to know if something is the same type
func (im *impl) As(err any) bool {
	_, ok := err.(CtxErr)
	return ok
}

// Is fulfills the interface to allow errors.Is
func (im *impl) Is(err error) bool { return im.As(err) }

// Context retrieves the context passed in when the error was created
func (im *impl) Context() context.Context { return im.ctx }

// Fields retrieves the fields from the context passed in when the error was created
func (im *impl) Fields() map[string]any { return Fields(im.ctx) }

// WithContext replaces the context of the error
func (im *impl) WithContext(ctx context.Context) { im.ctx = ctx }

// ** Helper Functions ** //

// SetHTTPStatusCode is equivelent to ctxerr.SetField(ctx, FieldKeyStatusCode, code)
func SetHTTPStatusCode(ctx context.Context, code int) context.Context {
	return global.SetHTTPStatusCode(ctx, code)
}
func (in Instance) SetHTTPStatusCode(ctx context.Context, code int) context.Context {
	return in.SetField(ctx, FieldKeyStatusCode, code)
}

// SetAction is equivelent to ctxerr.SetField(ctx, FieldKeyAction, action)
func SetAction(ctx context.Context, action string) context.Context {
	return global.SetAction(ctx, action)
}
func (in Instance) SetAction(ctx context.Context, action string) context.Context {
	return in.SetField(ctx, FieldKeyAction, action)
}

// SetCategory is equivelent to ctxerr.SetField(ctx, FieldKeyStatusCode, category)
func SetCategory(ctx context.Context, category any) context.Context {
	return global.SetCategory(ctx, category)
}
func (in Instance) SetCategory(ctx context.Context, category any) context.Context {
	return in.SetField(ctx, FieldKeyCategory, category)
}

// ** Hooks ** //

// DefaultLogHook is the default hook used log errors
// It is the fallback if there are no other handle hooks
func DefaultLogHook(err error) { global.DefaultLogHook(err) }
func (in Instance) DefaultLogHook(err error) {
	f := in.AllFields(err)
	b, merr := json.Marshal(f)
	fields := string(b)
	if merr != nil {
		fields = fmt.Sprintf("fields '%v' could not be marshalled as JSON: %s", f, merr)
	}
	log.Printf("%s - %s", err, fields)
}

// DefaultFieldsFunc is the default function to get fields from an error
func DefaultFieldsFunc(err error) map[string]any {
	if v, ok := err.(interface {
		Fields() map[string]any
	}); ok {
		return v.Fields()
	}
	return nil
}

// SetCodeHook takes the code and adds it to the context
func SetCodeHook(ctx context.Context, code string, wrapping error) context.Context {
	return global.SetCodeHook(ctx, code, wrapping)
}
func (in Instance) SetCodeHook(ctx context.Context, code string, wrapping error) context.Context {
	if code != "" {
		ctx = SetField(ctx, FieldKeyCode, code)
	}
	return ctx
}

// SetLocationHook get the location of where the error happened and adds it to the context
func SetLocationHook(ctx context.Context, code string, wrapping error) context.Context {
	return global.SetLocationHook(ctx, code, wrapping)
}
func (in Instance) SetLocationHook(ctx context.Context, code string, wrapping error) context.Context {
	ctx = SetField(ctx, FieldKeyLocation, CallerFunc(2))
	return ctx
}

/* HTTP helper function */

// NewHTTP creates a new error with action and status code
func NewHTTP(ctx context.Context, code, action string, statusCode int, message ...any) error {
	return global.NewHTTP(ctx, code, action, statusCode, message...)
}
func (in Instance) NewHTTP(ctx context.Context, code, action string, statusCode int, message ...any) error {
	if action != "" {
		ctx = SetAction(ctx, action)
	}
	if statusCode != 0 {
		ctx = SetHTTPStatusCode(ctx, statusCode)
	}
	return in.New(ctx, code, message...)
}

// NewHTTPf creates a new error  with action and status code and message formatting
func NewHTTPf(ctx context.Context, code, action string, statusCode int, message string, messageArgs ...any) error {
	return global.NewHTTPf(ctx, code, action, statusCode, message, messageArgs...)
}
func (in Instance) NewHTTPf(ctx context.Context, code, action string, statusCode int, message string, messageArgs ...any) error {
	if action != "" {
		ctx = in.SetAction(ctx, action)
	}
	if statusCode != 0 {
		ctx = in.SetHTTPStatusCode(ctx, statusCode)
	}
	return in.Newf(ctx, code, message, messageArgs...)
}

// WrapHTTP creates a new error with action and status code and another wrapped under it
func WrapHTTP(ctx context.Context, err error, code, action string, statusCode int, message ...any) error {
	return global.WrapHTTP(ctx, err, code, action, statusCode, message...)
}
func (in Instance) WrapHTTP(ctx context.Context, err error, code, action string, statusCode int, message ...any) error {
	if action != "" {
		ctx = in.SetAction(ctx, action)
	}
	if statusCode != 0 {
		ctx = in.SetHTTPStatusCode(ctx, statusCode)
	}
	return in.Wrap(ctx, err, code, message...)
}

// WrapHTTPf creates a new error with action and status code and a formatted message with another wrapped under it
func WrapHTTPf(ctx context.Context, err error, code, action string, statusCode int, message string, messageArgs ...any) error {
	return global.WrapHTTPf(ctx, err, code, action, statusCode, message, messageArgs...)
}
func (in Instance) WrapHTTPf(ctx context.Context, err error, code, action string, statusCode int, message string, messageArgs ...any) error {
	if action != "" {
		ctx = in.SetAction(ctx, action)
	}
	if statusCode != 0 {
		ctx = in.SetHTTPStatusCode(ctx, statusCode)
	}
	return in.Wrapf(ctx, err, code, message, messageArgs...)
}
