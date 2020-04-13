# ctxerr

[![GODOC](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/mvndaai/ctxerr)
[![Build
Status](https://travis-ci.org/mvndaai/ctxerr.svg?branch=master)](https://travis-ci.org/mvndaai/ctxerr/)

Go package for handling errors with extra context.


There are many ways to use the package and much more configuration in the godoc so please read it.


# Example HTTP Server
Here is a basic example on how it can be used in an HTTP package

The code below would make sure that all errors are logged and displayed in the same way.

## Code

```golang
import (
    ...
    "github.com/mvndaai/ctxerr"
    ch "github.com/mvndaai/ctxerr/http"
)

func handleError(w http.ResponseWriter, err error) {
    statusCode, response := ch.StatusCodeAndResponse(err, config.ShowErrorMessage, config.ShowErrorFields)
	w.WriteHeader(statusCode)
    w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}


func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    ...
    if err := userPermitted(ctx, user) err != nil {
        handleError(w, ctxerr.QuickWrap(ctx, err))
        return
    }

    if err := dbLookup(ctx, key); err != nil {
        handleError(w, ctxerr.QuickWrap(ctx, err))
        return
    }
    ...
}


func checkPermission(ctx context.Context, u user, permission string) error {
    if !u.HasPermission(permission) {
        ctx = ctxerr.SetField(ctx, "permission", permission)
        return ctxerr.New(ctx, "error_code_0", "permission not found")
    }
}


func userPermitted(ctx context.Context u user) error {
    ...
    for _, p := range permissions {
        if err := checkPermission(ctx, u, permission); err != nil {
            ctx = ctxerr.SetHTTPStatusCode(ctx, http.StatusUnauthorized)
            ctx = ctxerr.SetAction(ctx, fmt.Sprintf("Please request permission '%s' from an admin", permission))
            return ctxerr.QuickWrap(ctx, err)
        }
    }
    return nil
}


func dbLookup(ctx context.Context, key string) (string, error) {
    ctx = ctxerr.SetFields(ctx, "key", key)
    if !db.Has(key) {
        ctx = ctxerr.SetHTTPStatusCode(ctx, http.StatusNotFound)
        return "", ctxerr.New(ctx, "error_code_1", "key not found in db")
    }

    value, err := db.Get(key)
    if err != nil {
        return "", ctxerr.Wrap(ctx, err, "error_code_2", "db get error")
    }
}
```

## Responses
Given default configuration here is what could be returned.


### Missing permission

Status code `401`

HTTP response
```json
{
	"error": {
		"code" : "error_code_0",
		"action" : "Please request permission 'view' from an admin",
	}
}
```

Log
> handler : userPermitted : permission not found - {"permission": "view", ... }

### Missing Key

Status code `404`

HTTP response
```json
{
	"error": {
		"code" : "error_code_1",
	}
}
```

Log
> handler : dbLookup : key not found in db - {"key": "key", ... }


### DB Get Error

Status code `500`

HTTP response
```json
{
	"error": {
		"code" : "error_code_2",
	}
}
```

Log
> handler : dbLookup : db get error : *wrapped error* - {"key": "key", ... }
