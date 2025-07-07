package main

import (
	"math/rand"
)

func GetRandomFileName(prefix string, suffix string, middle string) string {
	ans := prefix
	for i := 0; i < 10; i++ {
		ans += string(rune('a' + rand.Intn(26)))
	}
	ans += middle
	return ans + suffix
}
