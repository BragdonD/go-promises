package gopromises

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// PromiseState represents the state of a promise.
type PromiseState string

// Constants for PromiseState.
const (
	PromisePendingState PromiseState = "PENDING" // Promise is pending.
	PromiseFulFillState PromiseState = "FULFILL" // Promise is fulfilled.
	PromiseRejectState  PromiseState = "REJECT"  // Promise is rejected.
)

// Promise represents an asynchronous operation that may produce a result of type T.
type Promise[T any] struct {
	// ctx is the context to manage promise lifecycle and cancellation.
	ctx context.Context
	// state is the current state of the promise.
	state PromiseState
	// once ensures resolve or reject is called only once.
	once sync.Once
	// sync is a channel to synchronize promise resolution or rejection.
	sync chan struct{}
	// err is the error if the promise is rejected.
	err error
	// res is the result if the promise is fulfilled.
	res *T
}

// resolve fulfills the promise with the given value.
func (p *Promise[T]) resolve(val T) {
	p.once.Do(func() {
		p.res = &val
		close(p.sync)
	})
}

// reject rejects the promise with the given error.
func (p *Promise[T]) reject(err error) {
	p.once.Do(func() {
		p.err = err   // Store the error.
		close(p.sync) // Close the sync channel to signal rejection.
	})
}

// NewPromise creates a new promise and immediately starts its execution.
// The executor function receives resolve and reject functions to settle the promise.
func NewPromise[T any](executor func(resolve func(T), reject func(error))) *Promise[T] {
	promise := &Promise[T]{
		err:   nil,
		res:   nil,
		state: PromisePendingState,
		sync:  make(chan struct{}),
		once:  sync.Once{},
		ctx:   context.Background(),
	}
	// Launch the Promise execution.
	go func() {
		executor(promise.resolve, promise.reject)
	}()
	return promise
}

// NewPromiseWithTimeout creates a new promise with a timeout and immediately starts its execution.
// The executor function receives resolve and reject functions to settle the promise.
func NewPromiseWithTimeout[T any](timeout time.Duration, executor func(resolve func(T), reject func(error))) *Promise[T] {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	promise := &Promise[T]{
		err:   nil,
		res:   nil,
		state: PromisePendingState,
		sync:  make(chan struct{}),
		once:  sync.Once{},
		ctx:   ctx,
	}
	// Launch the Promise execution.
	go func() {
		defer cancel()
		executor(promise.resolve, promise.reject)
	}()
	return promise
}

// NewPromiseWithContext creates a new promise with a given context and immediately starts its execution.
// The executor function receives resolve and reject functions to settle the promise.
func NewPromiseWithContext[T any](ctx context.Context, executor func(resolve func(T), reject func(error))) *Promise[T] {
	cctx, cancel := context.WithCancel(ctx)
	promise := &Promise[T]{
		err:   nil,
		res:   nil,
		state: PromisePendingState,
		sync:  make(chan struct{}),
		once:  sync.Once{},
		ctx:   cctx,
	}
	// Launch the Promise execution.
	go func() {
		defer cancel()
		executor(promise.resolve, promise.reject)
	}()
	return promise
}

// Function is a simple type to be able to pass function with or without return type
type Function interface{}

// isFunc returns a boolean indicating if obj is a function object
func isFunc(obj interface{}) bool {
	if obj == nil {
		return false
	}
	v := reflect.ValueOf(obj)
	return v.Kind() == reflect.Func
}

// Then adds handlers to be called when the promise is resolved or rejected.
// It returns a new promise that resolves or rejects based on the handlers' results.
//
// Parameters:
//   - resolved: a function to be called with the result when the promise is resolved.
//   - rejected: (optional) a variadic parameter of functions to be called with the error
//     when the promise is rejected. If provided, the first function is used.
//
// Returns:
//   - *Promise[any]: a pointer to a new Promise that resolves or rejects based on the handlers' results.
func (p *Promise[T]) Then(resolved Function, rejected ...func(error)) *Promise[any] {
	return NewPromise(func(resolve func(any), reject func(error)) {
		var rejectedFunc func(error)
		if len(rejected) > 0 {
			rejectedFunc = rejected[0]
		}

		select {
		case <-p.ctx.Done():
			handleRejection(p.ctx.Err(), rejectedFunc, reject)
		case <-p.sync:
			handleSync(p, resolved, resolve, reject, rejectedFunc)
		}
	})
}

// handleRejection is an helper function which handles the case where a promise result ends up being rejected
func handleRejection(err error, rejectedFunc func(error), reject func(error)) {
	if err != nil && rejectedFunc != nil {
		rejectedFunc(fmt.Errorf("promise rejected: %w", err))
	}
	reject(err)
}

// handleSync is a helper function which handles the case where a promise result ends
func handleSync[T any](p *Promise[T], resolved Function, resolve func(any), reject func(error), rejectedFunc func(error)) {
	if p.err != nil {
		handleRejection(p.err, rejectedFunc, reject)
		return
	}

	vresolved := reflect.ValueOf(resolved)
	if !isFunc(resolved) {
		reject(fmt.Errorf("unsupported func type: %T", resolved))
		return
	}

	resolvedType := vresolved.Type()
	if isValidResolvedFunc(p, resolvedType) {
		callResolvedFunc(p, vresolved, resolvedType, resolve, reject)
	} else {
		reject(fmt.Errorf("unsupported func type: %v, expected argument of type %v", resolvedType, reflect.TypeOf(*p.res)))
	}
}

// isValidResolvedFunc checks if a function is valid resolved function for the [Then] function
// A valid function should only have :
// - 1 or 0 entering parameter
func isValidResolvedFunc[T any](p *Promise[T], resolvedType reflect.Type) bool {
	return (resolvedType.NumIn() == 1 && resolvedType.In(0) == reflect.TypeOf(*p.res)) || resolvedType.NumIn() == 0
}

// callResolvedFunc is a helper function which handles the case where a promise is resolved
func callResolvedFunc[T any](p *Promise[T], vresolved reflect.Value, resolvedType reflect.Type, resolve func(any), reject func(error)) {
	var results []reflect.Value
	if resolvedType.NumIn() == 1 {
		results = vresolved.Call([]reflect.Value{reflect.ValueOf(*p.res)})
	} else {
		results = vresolved.Call(nil)
	}

	switch len(results) {
	case 0:
		resolve(nil)
	case 1:
		resolve(results[0].Interface())
	default:
		reject(fmt.Errorf("more than one output is not permitted, try to create an output struct"))
	}
}

// Catch attaches a handler for when the promise is rejected. If the promise is rejected,
// the provided onRejected function is called with the error.
//
// Parameters:
//   - onRejected: a function to be called with the error when the promise is rejected.
//
// Returns:
//   - *Promise[any]: a pointer to a new Promise that resolves with nil if the original promise is fulfilled,
//     or rejects if the original promise is rejected.
func (p *Promise[T]) Catch(onRejected func(error)) *Promise[any] {
	return NewPromise(func(resolve func(any), reject func(error)) {
		select {
		case <-p.ctx.Done():
			if p.ctx.Err() != nil || p.err != nil {
				reject(p.err)
				onRejected(p.err)
				return
			}
		case <-p.sync:
			if p.err != nil {
				onRejected(p.err)
				return
			}
			resolve(nil)
		}
	})
}

// Finally attaches a handler that is called when the promise is settled (either resolved or rejected).
// It returns a new promise that resolves with the original promise's result or rejects with its error.
// A Finally calls will always be executed even when the previous promise rejects.
//
// Parameters:
//   - onFinally: a function to be called when the promise is settled.
//
// Returns:
//   - *Promise[any]: a pointer to a new Promise that resolves with the original promise's result or
//     rejects with its error.
func (p *Promise[T]) Finally(onFinnaly func()) *Promise[any] {
	return NewPromise(func(resolve func(any), reject func(error)) {
		defer onFinnaly()
		select {
		case <-p.ctx.Done():
			if p.ctx.Err() != nil || p.err != nil {
				reject(p.err)
				return
			}
		case <-p.sync:
			if p.err != nil {
				reject(p.err)
				return
			}
			resolve(p.res)
		}
	})
}

// Await waits for the promise to be resolved or rejected and returns the result
// or error. It blocks until the promise is completed.
//
// Returns:
//   - *T: a pointer to the result of the promise if it resolves successfully.
//   - error: an error if the promise is rejected or the context is done.
func (p *Promise[T]) Await() (*T, error) {
	select {
	case <-p.ctx.Done():
		if p.ctx.Err() != nil {
			return nil, p.err
		}

	case <-p.sync:
		if p.err != nil {
			return nil, p.err
		} else {
			return p.res, nil
		}
	}
	return p.res, p.err
}

// All takes a variable number of promises and returns a new promise that resolves
// with a slice containing the results of all the input promises, or rejects if any
// of the input promises reject.
//
// The function type parameter T can be of any type.
//
// Parameters:
//   - promises: a variadic parameter of pointers to Promise objects.
//
// Returns:
//   - *Promise[[]T]: a pointer to a Promise that resolves with a slice of results
//     of type T or rejects with an error if any of the input promises reject.
func All[T any](promises ...*Promise[T]) *Promise[[]T] {
	return NewPromise(func(resolve func([]T), reject func(error)) {
		results := make([]T, 0)

		for _, promise := range promises {
			res, err := promise.Await()
			if err != nil {
				// If any promise rejects, reject the returned promise.
				reject(err)
				return
			}
			results = append(results, *res)
		}
		// Resolve the returned promise with the results slice.
		resolve(results)
	})
}
