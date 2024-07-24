package gopromises

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"
)

type PromiseState string

const (
	PromisePendingState PromiseState = "PENDING"
	PromiseFulFillState PromiseState = "FULLFILL"
	PromiseRejectState  PromiseState = "REJECT"
)

type Promise[T any] struct {
	ctx   context.Context
	state PromiseState
	once  sync.Once
	sync  chan struct{}
	err   error
	res   *T
}

func (p *Promise[T]) resolve(val T) {
	p.once.Do(func() {
		p.res = &val
		close(p.sync)
	})
}

func (p *Promise[T]) reject(err error) {
	p.once.Do(func() {
		p.err = err
		close(p.sync)
	})
}

func NewPromise[T any](executor func(resolve func(T), reject func(error))) *Promise[T] {
	promise := &Promise[T]{
		err:   nil,
		res:   nil,
		state: PromisePendingState,
		sync:  make(chan struct{}),
		once:  sync.Once{},
		ctx:   context.Background(),
	}
	// Launch the Promise execution
	go func() {
		executor(promise.resolve, promise.reject)
	}()
	return promise
}

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
	// Launch the Promise execution
	go func() {
		defer cancel()
		executor(promise.resolve, promise.reject)
	}()
	return promise
}

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
	// Launch the Promise execution
	go func() {
		defer cancel()
		executor(promise.resolve, promise.reject)
	}()
	return promise
}

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

// handleRejection is an helper function which handles the case where a promise result ends up beeing rejected
func handleRejection(err error, rejectedFunc func(error), reject func(error)) {
	if err != nil && rejectedFunc != nil {
		rejectedFunc(fmt.Errorf("promise rejected: %w", err))
	}
	reject(err)
}

// handleSync is an helper function which handles the case where a promise result ends up beeing accepted
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
