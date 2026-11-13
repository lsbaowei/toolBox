package durationring

import (
	"runtime"
	"time"
)

// cleanupYieldEvery 连续从 m1 删除这么多个 key 后让出一次调度。
// 全量清理只由一个协程顺序执行，这里再把删除打散，避免长时间独占修改 m1。
const cleanupYieldEvery = 64

func (d *DurationRing) cleanupLoop() {
	defer close(d.doneCh)

	// 启动时先完整扫描一轮，回收进程重启期间残留的过期索引。
	d.cleanupAllBuckets()

	ticker := time.NewTicker(d.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopCh:
			return
		case <-ticker.C:
			d.cleanupAllBuckets()
		}
	}
}

// cleanupAllBuckets 由清理协程顺序扫过全部物理 Bucket。
//
// 不为每个 Bucket 启动清理协程。并行删除会同时改 m1，抬高 sync.Map 的写同步成本，
// 业务 Load 即使不加锁也会变慢。
func (d *DurationRing) cleanupAllBuckets() {
	for slot := 0; slot < d.bucketCount; slot++ {
		select {
		case <-d.stopCh:
			return
		default:
		}
		d.cleanupBucket(slot)
		runtime.Gosched()
	}
}

func (d *DurationRing) cleanupBucket(slot int) {
	now := d.nowNano()
	b := &d.buckets[slot]
	deletedFromM1 := 0

	b.items.Range(func(key, value any) bool {
		// 上一轮已经释放分片锁。连续删除 m1 后让出调度，让业务 Load 先跑。
		if deletedFromM1 >= cleanupYieldEvery {
			runtime.Gosched()
			deletedFromM1 = 0
		}

		meta, ok := value.(indexMeta)
		if !ok {
			b.items.Delete(key)
			return true
		}

		lock := d.getKeyLock(key)
		lock.Lock()
		defer lock.Unlock()

		curRaw, ok := b.items.Load(key)
		if !ok {
			return true
		}
		curMeta, ok := curRaw.(indexMeta)
		if !ok || curMeta != meta {
			return true
		}

		// 索引尚未到期时不读、不写 m1。未到期的旧索引留到它自己的 ExpireAt 之后再核对。
		if now < curMeta.expireAt {
			return true
		}

		raw, exists := d.m1.Load(key)
		if !exists {
			compareAndDeleteMap(&b.items, key, meta)
			return true
		}
		entry, ok := raw.(*Entry)
		if !ok || entry == nil {
			compareAndDeleteMap(&b.items, key, meta)
			return true
		}

		// 旧索引：当前 Entry 已不属于该物理槽，或绝对槽/过期时间已变更。
		if entry.BucketSlot != slot ||
			entry.AbsoluteSlot != meta.absoluteSlot ||
			entry.ExpireAt != meta.expireAt {
			compareAndDeleteMap(&b.items, key, meta)
			return true
		}

		if now >= entry.ExpireAt {
			compareAndDeleteMap(&d.m1, key, entry)
			compareAndDeleteMap(&b.items, key, meta)
			deletedFromM1++
		}
		return true
	})
}
