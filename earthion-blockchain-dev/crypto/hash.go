package crypto

import "crypto/sha256"

func Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

func DoubleHash(data []byte) []byte {
	first := Hash(data)
	return Hash(first)
}