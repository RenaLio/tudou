package main

import (
	"fmt"
	"math/rand/v2"
)

type ctxKeyType struct{}

func main() {
	key1 := ctxKeyType{}
	key2 := ctxKeyType{}
	fmt.Println("key1 == key2:", key1 == key2)
	nums1, nums2 := 0, 0
	for _ = range 100 {
		randNum := rand.IntN(101)
		if randNum >= 90 {
			nums1++
		} else {
			nums2++
		}
	}
	fmt.Println("nums1:", nums1, "nums2:", nums2)
}
