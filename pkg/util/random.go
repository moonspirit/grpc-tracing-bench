package util

import (
	"math/rand"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

var randomBuffer = RandStringRunes(1000 * 1000)

func RandomBytes(length int) []byte {
	if length >= len(randomBuffer)-100 {
		return []byte(randomBuffer)
	}

	pos := rand.Intn(len(randomBuffer) - length - 1)
	return []byte(randomBuffer[pos : pos+length])
}
