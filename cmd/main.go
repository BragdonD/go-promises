package main

import (
	"fmt"

	gopromises "github.com/BragdonD/go-promises"
)

func main() {
	fmt.Println("Not resolved")
	val, err := gopromises.NewPromise[string](func(resolve func(string), reject func(error)) {
		resolve("resolved!")
	}).Then(func(val string) {
		fmt.Printf("Firstly resolved: %s\n", val)
	}).Then(func() {
		fmt.Printf("Secondly resolved\n")
	}).Then(func() string {
		return "test"
	}).Await()

	fmt.Println(*val, err)
}
