package minutering

import "time"

func (m *MinuteRing) cleanupLoop() {
	defer close(m.doneCh)

	if !m.skipStartupSweep {
		for slot := 0; slot < BucketCount; slot++ {
			m.cleanupBucket(slot)
		}
	}

	// 桶宽等于清理间隔。此刻在第 S 格，下一拍（一个间隔之后）时间进入 S+1，就清 S+1。
	m.nextSlot = bucketSlot(m.slotID(m.nowNano()) + 1)

	ticker := time.NewTicker(m.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			// 每拍只清一格，然后顺序加一。无阻塞时这格就是当前时间槽。
			// 停顿后不跳格：9 号清完卡住，恢复后清 10 号。
			m.cleanupBucket(m.nextSlot)
			m.nextSlot = (m.nextSlot + 1) % BucketCount
		}
	}
}

func (m *MinuteRing) cleanupBucket(slot int) {
	now := m.nowNano()
	b := &m.buckets[slot]

	b.items.Range(func(key, value any) bool {
		expireAt, ok := value.(int64)
		if !ok {
			b.items.Delete(key)
			return true
		}

		lock := m.getKeyLock(key)
		lock.Lock()
		defer lock.Unlock()

		curExpire, ok := b.items.Load(key)
		if !ok {
			return true
		}
		curExpireAt, ok := curExpire.(int64)
		if !ok || curExpireAt != expireAt {
			return true
		}

		raw, exists := m.m1.Load(key)
		if !exists {
			compareAndDeleteMap(&b.items, key, expireAt)
			return true
		}
		entry, ok := raw.(*Entry)
		if !ok || entry == nil {
			compareAndDeleteMap(&b.items, key, expireAt)
			return true
		}

		// 当前 Entry 已不属于该 Bucket，说明是 TTL 更新前的旧索引。
		if entry.BucketSlot != slot || entry.ExpireAt != expireAt {
			compareAndDeleteMap(&b.items, key, expireAt)
			return true
		}

		if now >= entry.ExpireAt {
			compareAndDeleteMap(&m.m1, key, entry)
			compareAndDeleteMap(&b.items, key, expireAt)
		}
		return true
	})
}
