package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// RandomHex 返回 n 字节随机数的十六进制字符串（长度为 2n）。
func RandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
