package main

import "fmt"

func main() {
	fmt.Println(a(), "==", 10)
	fmt.Println(b(), "==", 10)
}

func a() int {
	defer func() { returnFrom(a, 10) }()
	return 5
}

func wrapper(fn func() int) int {
	defer func() { returnFrom(wrapper, 5) }()
	return fn()
}
func b() int {
	return wrapper(func() int {
		returnFrom(wrapper, 10)
		return 5
	})
}
