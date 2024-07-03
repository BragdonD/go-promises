package gopromises_test

import (
	"fmt"
	"testing"

	gopromises "github.com/BragdonD/go-promises"
)

func TestPromise(t *testing.T) {
	fmt.Println("Not resolved")
	gopromises.NewPromise[string](func(resolve func(string), reject func(error)) {
		resolve("resolved!")
	}).Then(func(val string) string {
		fmt.Printf("Definitely resolved: %s\n", val)
		return fmt.Sprintf("Definitely resolved: %s\n", val)
	}).Then(func(val string) {
		fmt.Printf("Finally: %s", val)
	}).Await()
}
