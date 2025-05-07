package utils_random

import (
	"math/rand"
	"sync"
	"time"
)

/*

以下是一个简单实用的 Golang 随机工具包，封装了推荐用法，并支持：
	•	创建独立的随机源（线程安全）
	•	生成整数、浮点数、随机字符串等
	•	可选指定种子（方便测试）
*/

type RandUtil struct {
	r  *rand.Rand
	mu sync.Mutex
}

// New creates a new RandUtil instance with a random seed.
func New() *RandUtil {
	seed := time.Now().UnixNano()
	return NewWithSeed(seed)
}

// NewWithSeed creates a new RandUtil instance with the given seed.
func NewWithSeed(seed int64) *RandUtil {
	src := rand.NewSource(seed)
	return &RandUtil{r: rand.New(src)}
}

// Intn returns a random int in [0, n).
func (ru *RandUtil) Intn(n int) int {
	ru.mu.Lock()
	defer ru.mu.Unlock()
	return ru.r.Intn(n)
}

// Float64 returns a random float64 in [0.0, 1.0).
func (ru *RandUtil) Float64() float64 {
	ru.mu.Lock()
	defer ru.mu.Unlock()
	return ru.r.Float64()
}

// String generates a random alphanumeric string of given length.
func (ru *RandUtil) String(length int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	ru.mu.Lock()
	defer ru.mu.Unlock()
	for i := range result {
		result[i] = letters[ru.r.Intn(len(letters))]
	}
	return string(result)
}
