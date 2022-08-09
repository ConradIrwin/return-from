package main

import "fmt"

func Range[T any](s []T, fn func(T)) {
	for _, v := range s {
		fn(v)
	}
}

func main() {
	fmt.Println(a(), "==", 10)
	fmt.Println(b[string]())
}

func a() int {
	Range([]int{1, 2, 3, 4}, func(v int) {
		if v%2 == 0 {
			returnFrom(Range[int])
		}
	})
	return 0
}

func b[T any]() T {
	returnFrom(b[string], *new(T))
	panic("unreachable")
}
