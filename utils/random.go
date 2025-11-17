package utils

import (
	"math/rand"
	"strings"
)

func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]string, length)
	for i := 0; i < length; i++ {
		b = append(b, string(charset[rand.Intn(len(charset))]))
	}
	return strings.Join(b, "")
}
