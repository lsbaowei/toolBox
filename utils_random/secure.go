package utils_random

import (
	crand "crypto/rand"
	"math/big"
)

// SecureInt64 返回 [0, 2^63) 的密码学安全随机 int64。
func SecureInt64() (int64, error) {
	upper := new(big.Int).Lsh(big.NewInt(1), 63)
	n, err := crand.Int(crand.Reader, upper)
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}

// SecureIntn 返回 [0, max) 的密码学安全随机整数；max <= 0 时返回 0。
func SecureIntn(max int64) (int64, error) {
	if max <= 0 {
		return 0, nil
	}
	n, err := crand.Int(crand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}

// IntWithSafety 使用 crypto/rand，返回 [0, 2^63) 范围内的 int64。
func IntWithSafety() (int64, error) {
	return SecureInt64()
}
