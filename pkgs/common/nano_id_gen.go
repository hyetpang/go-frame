package common

import gonanoid "github.com/matoous/go-nanoid/v2"

func GenNanoID() (string, error) {
	return gonanoid.Generate("abcdefghijklmnopqrstuvwxyz", 10)
}
