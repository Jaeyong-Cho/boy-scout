package assertutil

import "fmt"

// Assertf panics with the formatted message if cond is false.
func Assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}
