package minutering

import "time"

func (m *MinuteRing) cleanupLoop() {
	defer close(m.doneCh)

	if !m.skipStartupSweep {
		for slot := 0; slot < BucketCount; slot++ {
			m.cleanupBucket(slot)
		}
		m.mu.Lock()
		m.lastCleanedMinute = minuteID(m.nowNano())
		m.mu.Unlock()
	}

	ticker := time.NewTicker(m.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.cleanupTickOnce()
		}
	}
}

// cleanupTickOnce 按绝对分钟推进清理；若上次延迟，补扫遗漏 Bucket。
// 同一时刻只允许一个清理任务运行。
func (m *MinuteRing) cleanupTickOnce() {
	m.mu.Lock()
	if m.cleaning {
		m.mu.Unlock()
		return
	}
	m.cleaning = true
	currentMinute := minuteID(m.nowNano())
	last := m.lastCleanedMinute
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.cleaning = false
		m.lastCleanedMinute = currentMinute
		m.mu.Unlock()
	}()

	if last < 0 {
		m.cleanupBucket(bucketSlot(currentMinute))
		return
	}

	// 补扫 (last, currentMinute]；若停顿超过一整轮，至少扫满 60 个物理槽。
	delta := currentMinute - last
	if delta <= 0 {
		return
	}
	if delta > BucketCount {
		delta = BucketCount
	}
	for i := int64(1); i <= delta; i++ {
		m.cleanupBucket(bucketSlot(last + i))
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
