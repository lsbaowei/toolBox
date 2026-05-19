package utils_random

import (
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

// IntV2 返回伪随机 int，复用包内 *rand.Rand。
func IntV2() int {
	return pseudoInt()
}

// IntWithT 想要可重复的随机数（调试/测试用）
func IntWithT(seed int64) int {
	r := rand.New(rand.NewSource(seed))
	return r.Int()
}
