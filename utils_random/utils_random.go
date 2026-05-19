package utils_random

import (
	rand2 "crypto/rand"
	"math/big"
	"math/rand"
	"time"
)

// Int 使用全局随机源；每次调用会 Seed，已废弃，请用 IntV2 或 RandUtil。
//
// Deprecated: 使用 IntV2 或 utils_random.New。
func Int() int {
	rand.Seed(time.Now().UnixNano()) //nolint:staticcheck // 保持旧行为
	return rand.Int()
}

// IntV2 普通版本的升级，替换掉了 Seed
func IntV2() int {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))

	return r.Int()
}

// IntWithSafety 使用 crypto/rand，返回 [0, 2^63) 范围内的 int64。
func IntWithSafety() (int64, error) {
	upper := new(big.Int).Lsh(big.NewInt(1), 63)
	n, err := rand2.Int(rand2.Reader, upper)
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}

// IntWithT 想要可重复的随机数（调试/测试用）
func IntWithT(seed int64) int {
	r := rand.New(rand.NewSource(seed)) //  seed 固定种子
	return r.Int()
}
