package tests

import (
	"math/rand"
	"unsafe"

	"github.com/google/uuid"
)

var (
	Generator  generator
	validChars = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789[]{}=_!?<>!@#$%^&*() \t\n\r")
)

type generator struct{}

func (_ generator) UUID() string {
	return uuid.Must(uuid.NewRandom()).String()
}

// Generate a random string
// No arguments: 0-200 length
// Single integer: exactly N length
// Two integers: between A and B lengths
func (g generator) String(constraints ...int) string {
	switch len(constraints) {
	case 0:
		return g.String(rand.Intn(200))
	case 1:
		l := constraints[0]
		str := make([]byte, l)
		for i := 0; i < l; i++ {
			str[i] = validChars[rand.Intn(len(validChars))]
		}
		return *(*string)(unsafe.Pointer(&str))
	case 2:
		min := constraints[0]
		max := constraints[1]
		return g.String(rand.Intn(max-min+1) + min)
	default:
		panic("String() should take 0 (random), 1 (exact length) or 2 (between A and B length) integers")
	}
}
