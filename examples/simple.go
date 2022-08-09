package main

import "fmt"

func main() {
	fmt.Println(a(), "==", 10)
}

func a() int {
	func() { returnFrom(a, 10) }()
	return 5
}
