package vcode

import "math/rand"

// Base58Chars is the Bitcoin Base58 alphabet used by RandomBase58.
const Base58Chars = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// RandomBase58 returns a pseudorandom Base58 string of length.
// It is not intended for cryptographic secrets.
func RandomBase58(length int) string {
	letters := []rune(Base58Chars)
	chars := make([]rune, length)
	for i := range chars {
		chars[i] = letters[rand.Intn(len(letters))]
	}
	return string(chars)
}
