package utils_random

import (
	"math/rand"
	"sync"
	"time"
)

var (
	pseudoMu sync.Mutex
	pseudoR  = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func pseudoInt() int {
	pseudoMu.Lock()
	pseudoR.Seed(time.Now().UnixNano())
	n := pseudoR.Int()
	pseudoMu.Unlock()
	return n
}
