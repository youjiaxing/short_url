package main

import (
	"math/rand"
	"time"
)

var (
	randomBytes = []byte(`123456789abcdefghijklmnpqrstuvwxyzABCDEFGHIJKLMNPQRSTUVWXYZ`)
	random      = rand.New(rand.NewSource(time.Now().UnixNano()))

	SimpleGen = SimpleGenerator(simpleGeneratorFunc)
)

type SimpleGenerator func(int) string

func (g SimpleGenerator) gen(length int) string {
	return g(length)
}

func simpleGeneratorFunc(length int) string {
	bytesLen := len(randomBytes)

	buf := make([]byte, length)
	for i := 0; i < length; i++ {
		buf[i] = randomBytes[random.Intn(bytesLen)]
	}

	return string(buf)
}
