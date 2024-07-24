package main

import (
	"fmt"
	"log"

	gopromises "github.com/BragdonD/go-promises"
)

func main() {
	log.Println("Not resolved")

	test, err := gopromises.NewPromise(func(resolve func(string), reject func(error)) {
		reject(fmt.Errorf("a problem happened"))
		// resolve("resolved!")
	}).Then(func(val string) {
		log.Printf("Firstly resolved: %s\n", val)
	}).Finally(func() {
		log.Print("Finally")
	}).Then(func() {
		log.Printf("Secondly resolved\n")
	}).Then(func() string {
		return "test"
	}).Catch(func(err error) {
		log.Fatal(err)
	}).Await()

	if err != nil {
		panic(err)
	}

	fmt.Println(*test)
}
