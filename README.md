# ctxerr <span style='float: right;'>[![doc](https://img.shields.io/github/v/tag/mvndaai/ctxerr?include_prereleases&label=godoc&sort=semver)](https://pkg.go.dev/github.com/mvndaai/ctxerr)</span>


Go package for adding extra context to errors.


This implementaton of the [`error` interface](https://pkg.go.dev/builtin#error) adds fields (i.e. `map[string]any`) to a `context.Context` and attaches that to the returned error. There are a few fields defined by this package but most

```golang
if err != nil {
    ctx = ctxerr.SetField(ctx, "field", "value")
    return ctxerr.Wrap(ctx, err, "ERROR_CODE", "message")
}
```

If you pass the context through your functions then fields can be added when the the data first appears and are automatically available in deeper functions.

## Example of basic usage

In this example, even though `params` are set in a the top function, `foo`, they are available even in the `err` returned by the deepest function, `baz`.

```golang
func foo(req request) {
    ctx := req.Context()
    ctx = ctxerr.SetField(ctx, "params", req.Params)

    if err := baz(ctx); err != nil {
        err = ctxerr.QuickWrap(ctx, err)
        ctxerr.Handle(err)
        return
    }
}

func bar(ctx context.Context) error {
    // If err == nil; (Quick)Wrap will return nil
    return ctxerr.QuickWrap(ctx, baz(ctx))
}

func baz(ctx context.Context) error {
    return ctxerr.New(ctx, "NOT_IMPLEMENTED", "function not implemented")
}
```

## HTTP

There are helper functions `NewHTTP` and `WrapHTTP` that set fields inline for a status code and an 'action'. Actions are what this package calls external user facing messages. They are the onces that can be shown to users without revealing internal details of your application.

There is a subpackage [ctxerr/http](https://pkg.go.dev/github.com/mvndaai/ctxerr/http) that simplifies returning error JSON to an http request.




## Handle

Errors should [only be handled once](https://dave.cheney.net/practical-go/presentations/qcon-china.html#_only_handle_an_error_once). Rather than calling `log` at the topmost return call `ctxerr.Handle(err)` so all errors can be handled in the same way. This is especially helpful in `go func()` and `defer func()`.

## Hooks

Configuration is done through hooks.

[`AddCreateHook`](https://pkg.go.dev/github.com/mvndaai/ctxerr#AddCreateHook) adds hooks that are run on ever `New` or `(Quick)Wrap`. Builtin hooks add the error code and a location of the error to the context.

[`AddHandleHook`](https://pkg.go.dev/github.com/mvndaai/ctxerr#AddHandleHook) adds hooks that are run on `Handle`. If no hooks exist it will run a [default log hook](https://pkg.go.dev/github.com/mvndaai/ctxerr#Instance.DefaultLogHook). Use this to create a hook to log consistently however you want or even create a metric on each error by code.

Common configurations might be available in the packages under [ctxerrhelper](https://github.com/mvndaai/ctxerrhelper). There each package has its own `go.mod` file to avoid adding extra dependencies to your service.
