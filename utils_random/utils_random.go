package utils_random

import (
	rand2 "crypto/rand"
	"math/big"
	"math/rand"
	"time"
)

// Int 普通
func Int() int {
	rand.Seed(time.Now().UnixNano()) // 一定要播种，否则每次结果一样
	return rand.Int()
}

// IntV2 普通版本的升级，替换掉了 Seed
func IntV2() int {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))

	return r.Int()
}

// IntWithSafety 安全 & 不可预测
func IntWithSafety() int64 {
	n, err := rand2.Int(rand2.Reader, big.NewInt(2^63)) // 0 <= n < 100
	if err != nil {
		panic(err)
	}
	return n.Int64()
}

// IntWithT 想要可重复的随机数（调试/测试用）
func IntWithT(seed int64) int {
	r := rand.New(rand.NewSource(seed)) //  seed 固定种子
	return r.Int()
}
