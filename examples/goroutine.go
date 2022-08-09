package main

import "fmt"

func main() {
	fmt.Println(a(), "==", 10)
}

func a() int {
	c := make(chan interface{})
	go func() {
		defer func() { c <- recover() }()
		returnFrom(a, 10)
	}()
	panic(<-c)
}
