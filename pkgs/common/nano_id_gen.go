package common

import (
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func GenNanoID() (string, error) {
	return gonanoid.Generate(alphaNumber, 10)
}

var (
	alphaLower  string = "abcdefghijklmnopqrstuvwxyz"
	alphaUpper  string = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	number      string = "1234567890"
	alphaNumber        = alphaLower + alphaUpper + number
)

func GenNanoIDFromAlphaNumber(size int) (string, error) {
	return gonanoid.Generate(alphaNumber, size)
}
