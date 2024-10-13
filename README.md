# Go-Promises

[![Build Status](https://github.com/bragdond/go-promises/workflows/Run%20Tests/badge.svg?branch=master)](https://github.com/bragdond/go-promises/actions?query=branch%3Amaster)
[![codecov](https://codecov.io/gh/bragdond/go-promises/branch/master/graph/badge.svg)](https://codecov.io/gh/bragdond/go-promises)
[![Go Report Card](https://goreportcard.com/badge/github.com/bragdond/go-promises)](https://goreportcard.com/report/github.com/bragdond/go-promises)
[![Go Reference](https://pkg.go.dev/badge/github.com/bragdond/go-promises?status.svg)](https://pkg.go.dev/github.com/bragdond/go-promises?tab=doc)
[![Sourcegraph](https://sourcegraph.com/github.com/bragdond/go-promises/-/badge.svg)](https://sourcegraph.com/github.com/bragdond/go-promises?badge)
[![Open Source Helpers](https://www.codetriage.com/bragdond/go-promises/badges/users.svg)](https://www.codetriage.com/bragdond/go-promises)
[![Release](https://img.shields.io/github/release/bragdond/go-promises.svg?style=flat-square)](https://github.com/bragdond/go-promises/releases)
[![TODOs](https://badgen.net/https/api.tickgit.com/badgen/github.com/bragdond/go-promises)](https://www.tickgit.com/browse?repo=github.com/bragdond/go-promises)[![Last Commit](https://img.shields.io/github/last-commit/avelino/awesome-go)](https://img.shields.io/github/last-commit/avelino/awesome-go)

**Go-Promises** is a lightweight Go library that ports **JavaScript-like** Promises to Go. It allows handling ``asynchronous`` operations in Go with familiar methods such as ``Then``, ``Catch``, ``Finally``, and more. The library also supports promise chaining, timeouts, context management, and more.

**Go-promises's key features are**

- Handle **async operations** and manage success/failure using a familiar pattern.
- **Chain** multiple promises.
- Create promises that can be **canceled** or **timed out**.
- Wait for **multiple promises** to resolve with the All method.

## Getting started

### Prerequisites

Go-Promises requires [Go](https://go.dev/) version [1.22](https://go.dev/doc/devel/release#go1.22.0) or above.

### Installation

With [Go's module support](https://go.dev/wiki/Modules#how-to-use-modules), `go [build|run|test]` automatically fetches the necessary dependencies when you add the import in your code:

```sh
import "github.com/bragdond/go-promises"
```

Alternatively, use `go get`:

```sh
go get -u github.com/bragdond/go-promises
```

### Usage Example

Here's a simple example of how to use promises in Go with this library.

```go
package main

import (
    "errors"
    "fmt"
    "gopromises"
)

func main() {
    promise := gopromises.NewPromise(func(resolve func(string), reject func(error)) {
        // Simulate some async operation
        if success := true; success {
            resolve("Success!")
        } else {
            reject(errors.New("Failure"))
        }
    })

    promise.
        Then(func(result any) {
            fmt.Println("Resolved with:", result)
        }, func(err error) {
            fmt.Println("Rejected with error:", err)
        }).Catch(func(err error) {
            fmt.Println("Caught an error:", err)
        }).Finally(func() {
            fmt.Println("Promise is settled.")
        })
}
```

### Advanced Usage

#### Timeout Support

You can create promises with a timeout, and if the promise is not settled before the timeout, it will be rejected automatically.

```go
promise := gopromises.NewPromiseWithTimeout(time.Second * 2, func(resolve func(string), reject func(error)) {
    // Simulate long operation
    time.Sleep(3 * time.Second)
    resolve("This will timeout!")
})

promise.Catch(func(err error) {
    fmt.Println("Promise failed:", err)
})
```

#### Working with Context

If you want to tie a promise to a specific context (for cancellation or deadlines), use ``NewPromiseWithContext``.

```go
ctx, cancel := context.WithCancel(context.Background())
promise := gopromises.NewPromiseWithContext(ctx, func(resolve func(string), reject func(error)) {
    // Async operation
    resolve("Operation successful")
})

cancel() // Cancels the promise before it resolves

promise.Catch(func(err error) {
    fmt.Println("Promise canceled:", err)
})
```

#### Chaining Promises

You can chain multiple Then calls for asynchronous flow control:

```go
promise.
    Then(func(result any) {
        fmt.Println("First step:", result)
        return "Next step"
    }).
    Then(func(result any) {
        fmt.Println("Second step:", result)
    })
```

#### Awaiting Promises

To block and wait for a promise result synchronously, use Await.

```go
result, err := promise.Await()
if err != nil {
    fmt.Println("Promise failed:", err)
} else {
    fmt.Println("Promise result:", *result)
}
```

#### Handling Multiple Promises

You can wait for multiple promises to settle with All.

```go
p1 := gopromises.NewPromise(func(resolve func(string), reject func(error)) { resolve("Promise 1 resolved") })
p2 := gopromises.NewPromise(func(resolve func(string), reject func(error)) { resolve("Promise 2 resolved") })

allPromises := gopromises.All(p1, p2)

allPromises.Then(func(results any) {
    fmt.Println("All promises resolved with results:", results)
})
```

## Documentation

For full documentation, visit [godoc.org/github.com/bragdond/go-promises](godoc.org/github.com/bragdond/go-promises).

## License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.

## Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details on submitting patches and the contribution workflow.