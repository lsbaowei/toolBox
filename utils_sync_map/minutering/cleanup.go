package minutering

import "time"

func (m *MinuteRing) cleanupLoop() {
	defer close(m.doneCh)

	if !m.skipStartupSweep {
		for slot := 0; slot < BucketCount; slot++ {
			m.cleanupBucket(slot)
		}
	}

	ticker := time.NewTicker(m.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			// 按固定顺序每次只清一个桶，指针不跟绝对分钟跳。
			//
			// 9 号清完被卡住，恢复后下一拍清 10 号，再下一拍清 11 号。
			// 已经排到的桶按原节奏继续被清，该删的数据不用再等一整圈；
			// 每拍仍然只有一个桶，不会在恢复时一次性扫很多桶。
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
