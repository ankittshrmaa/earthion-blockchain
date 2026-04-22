package crypto

import (
	"math/big"
)

const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// Base58Encode encodes a byte slice to Base58
func Base58Encode(input []byte) []byte {
	if len(input) == 0 {
		return []byte{}
	}

	// Count leading zeros
	leadingZeros := 0
	for _, b := range input {
		if b == 0 {
			leadingZeros++
		} else {
			break
		}
	}

	// Convert to big int
	num := new(big.Int)
	num.SetBytes(input)

	// Convert to Base58
	var result []byte
	for num.Sign() != 0 {
		mod := new(big.Int)
		num.DivMod(num, big.NewInt(58), mod)
		result = append([]byte{base58Alphabet[mod.Int64()]}, result...)
	}

	// Add leading '1's for each leading zero byte
	for i := 0; i < leadingZeros; i++ {
		result = append([]byte{'1'}, result...)
	}

	return result
}

// Base58Decode decodes a Base58 string to bytes
func Base58Decode(input string) []byte {
	if len(input) == 0 {
		return []byte{}
	}

	// Count leading '1's
	leadingOnes := 0
	inputBytes := []byte(input)
	for _, c := range inputBytes {
		if c == '1' {
			leadingOnes++
		} else {
			break
		}
	}

	// Convert from Base58
	num := big.NewInt(0)
	for _, c := range inputBytes {
		// Find character in alphabet
		idx := -1
		for i, a := range base58Alphabet {
			if byte(a) == c {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil // Invalid character
		}

		tmp := big.NewInt(int64(idx))
		num.Mul(num, big.NewInt(58))
		num.Add(num, tmp)
	}

	result := num.Bytes()

	// Add leading zeros
	for i := 0; i < leadingOnes; i++ {
		result = append([]byte{0}, result...)
	}

	return result
}

// Checksum returns first 4 bytes of DoubleHash
func Checksum(data []byte) []byte {
	hash := DoubleHash(data)
	return hash[:4]
}

// AddChecksum appends checksum to data
func AddChecksum(data []byte) []byte {
	checksum := Checksum(data)
	return append(data, checksum...)
}

// VerifyChecksum verifies the checksum of data (last 4 bytes)
func VerifyChecksum(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	checksum := data[len(data)-4:]
	payload := data[:len(data)-4]

	expected := Checksum(payload)
	for i := 0; i < 4; i++ {
		if checksum[i] != expected[i] {
			return false
		}
	}
	return true
}
