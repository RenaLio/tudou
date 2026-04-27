package main

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/goccy/go-json"
)

type ctxKeyType struct{}

type Box struct {
	CreateAt time.Time `json:"createAt"`
}

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
	box := Box{
		CreateAt: time.Now(),
	}
	fmt.Println("box:", box)
	boxBytes, err := json.Marshal(box)
	if err != nil {
		fmt.Println("json marshal error:", err)
	}
	fmt.Println("boxBytes:", string(boxBytes))
}
