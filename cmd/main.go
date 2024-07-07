package main

import (
	"fmt"

	gopromises "github.com/BragdonD/go-promises"
)

func main() {
	fmt.Println("Not resolved")

	gopromises.NewPromise(func(resolve func(string), reject func(error)) {
		resolve("resolved!")
	}).Then(func(val string) {
		fmt.Printf("Firstly resolved: %s\n", val)
	}).Finally(func() {
		fmt.Println("Finally")
	}).Then(func() {
		fmt.Printf("Secondly resolved\n")
	}).Then(func() string {
		return "test"
	}).Await()
}
