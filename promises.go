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
	PromisePendingState = "PENDING"
	PromiseFulFillState = "FULLFILL"
	PromiseRejectState  = "REJECT"
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
func isFunc(obj reflect.Value) bool {
	// Zero value reflected: not a valid function
	if obj == (reflect.Value{}) {
		return false
	}

	if obj.Type().Kind() != reflect.Func {
		return false
	}

	return true
}

func (p *Promise[T]) Then(resolved Function, rejected ...func(error)) *Promise[any] {
	return NewPromise(func(resolve func(any), reject func(error)) {
		var rejectedFunc func(error)
		if len(rejected) > 0 {
			rejectedFunc = rejected[0]
		}
		select {
		case <-p.ctx.Done():
			if p.ctx.Err() != nil || p.err != nil {
				if rejectedFunc != nil {
					rejectedFunc(fmt.Errorf("promise rejected: %w", p.err))
				}
				reject(p.err)
				return
			}
		case <-p.sync:
			if p.err != nil {
				if rejectedFunc != nil {
					rejectedFunc(p.err)
					reject(p.err)
				}
				return
			}

			vresolved := reflect.ValueOf(resolved)
			if !isFunc(vresolved) {
				reject(fmt.Errorf("unsupported func type: %T", resolved))
				return
			}

			resolvedType := vresolved.Type()
			if resolvedType.NumIn() == 1 && resolvedType.In(0) == reflect.TypeOf(*p.res) {
				if vresolved.Type().NumOut() == 1 {
					result := vresolved.Call([]reflect.Value{reflect.ValueOf(*p.res)})[0].Interface()
					resolve(result)
				} else if vresolved.Type().NumOut() == 0 {
					vresolved.Call([]reflect.Value{reflect.ValueOf(*p.res)})
					resolve(nil)
				} else {
					reject(fmt.Errorf("more than one output is not permitted, try to create an output struct"))
				}
			} else if resolvedType.NumIn() == 0 {
				if vresolved.Type().NumOut() == 1 {
					result := vresolved.Call([]reflect.Value{})[0].Interface()
					resolve(result)
				} else if vresolved.Type().NumOut() == 0 {
					vresolved.Call([]reflect.Value{})
					resolve(nil)
				} else {
					reject(fmt.Errorf("more than one output is not permitted, try to create an output struct"))
				}
			} else {
				reject(fmt.Errorf("unsupported func type: %v, expected argument of type %v", resolvedType, reflect.TypeOf(*p.res)))
			}
		}
	})
}

func (p *Promise[T]) Catch() *Promise[T] {
	return p
}

func (p *Promise[T]) Finally() *Promise[T] {
	return p
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
