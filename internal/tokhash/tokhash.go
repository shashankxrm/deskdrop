package tokhash

import (
	"crypto/hmac"
	"crypto/sha256"
)

// Sum is HMAC-SHA256(pepper, plaintext). Used for device tokens and pairing codes at rest.
func Sum(pepper, plaintext string) []byte {
	mac := hmac.New(sha256.New, []byte(pepper))
	_, _ = mac.Write([]byte(plaintext))
	return mac.Sum(nil)
}
