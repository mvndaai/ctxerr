# ctxerr/http <span style='float: right;'>[![DOC](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/mvndaai/ctxerr/http)</span>

This is a package to use with `ctxerr` to simplify returning an error JSON object in a REST API.

The function `StatusCodeAndResponse` converts the error with its fields into a struct that can be returned and pull the status code out.

```go
import (
    ctxerrhttp "github.com/mvndaai/ctxerr/http"
)

func httpErrorHandler(w http.ResponseWriter, showMessage, showFields bool)
    // Handle the error just this once at the top of the stack
    ctxerr.Handle(err)

    statusCode, response := ctxerrhttp.StatusCodeAndResponse(err, showMessage, showFields)

    w.WriteHeader(statusCode)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

## JSON

Depending on if you how you configured the show booleans you will be returned something like these. Make sure to hide message and fields on normal requests in production to avoid revealing too many implemenation details to nefarious users.

```javascript
{
    "error": {
        "code" : "VALIDATION_ZIP",
        "action" : "Zip code was not a 5 digit number",
        "traceID" : "6eee7a47-a90b-445d-9b18-6b801e46d021",
    }
}
```

```javascript
{
    "error": {
        "code" : "VALIDATION_ZIP",
        "action" : "Zip code was not 5 digits",
        "messsage" : "validation error on zip code",
        "traceID" : "6eee7a47-a90b-445d-9b18-6b801e46d021",
        "fields" : {"zip": "927"},
    }
}
```

## Recommendations

* Use the helper functions `ctxerr.NewHTTP` and `ctxerr.WrapHTTP` at the deepest point in your code to add a status code and action when you know what the actual issue is.
* When using a logger that has `warn` and `error` in your `ctxerr.AddHandleHook` log errors with status code 4xx as a warning and everything else as an error.