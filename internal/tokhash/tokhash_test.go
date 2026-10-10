package tokhash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestSum_isHMAC(t *testing.T) {
	pepper := "pepper-value-for-tests"
	msg := "device-token"
	got := Sum(pepper, msg)
	mac := hmac.New(sha256.New, []byte(pepper))
	_, _ = mac.Write([]byte(msg))
	want := mac.Sum(nil)
	if !hmac.Equal(got, want) {
		t.Fatalf("got %s want %s", hex.EncodeToString(got), hex.EncodeToString(want))
	}
	concat := sha256.Sum256([]byte(pepper + msg))
	if hmac.Equal(got, concat[:]) {
		t.Fatal("must not be sha256(pepper||token)")
	}
}
