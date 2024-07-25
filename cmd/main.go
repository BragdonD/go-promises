package main

import (
	"fmt"
	"log"

	gopromises "github.com/BragdonD/go-promises"
)

func main() {
	log.Println("Not resolved")

	promise1 := gopromises.NewPromise(func(resolve func(string), reject func(error)) {
		//reject(fmt.Errorf("a problem happened"))
		resolve("resolved 1!")
	}).Then(func(val string) string {
		return val
	})

	promise2 := gopromises.NewPromise(func(resolve func(int), reject func(error)) {
		//reject(fmt.Errorf("a problem happened"))
		resolve(2)
	}).Then(func(val int) int {
		return val
	})

	val, err := gopromises.All(promise1, promise2).Await()
	if err != nil {
		panic(err)
	}

	fmt.Println(*val...)
}
