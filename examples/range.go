package main

import "fmt"

// make a fake "slices" package
var slices slicesImpl

type slicesImpl struct{}

func (slicesImpl) Range(s []int, fn func(int)) {
	for _, v := range s {
		fn(v)
	}
}

func main() {
	ret, err := a()
	fmt.Println(ret, "==", 10, err)
}

func a() (int, error) {
	s := []int{2, 4, 6, 8, 10, 12}
	slices.Range(s, func(i int) {
		if i%5 == 0 {
			returnFrom(a, i, nil)
		}
		if i > 10 {
			panic("unreachable")
		}
	})
	return 0, fmt.Errorf("not found")
}
