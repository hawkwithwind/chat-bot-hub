package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

func CheckSum(src []byte) []byte {
	h := sha256.New()
	h.Write(src)
	return h.Sum(nil)
}

func HexString(src []byte) string {
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)

	return string(dst)
}
