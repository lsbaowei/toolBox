



# Go Cache 设计规格：双 sync.Map + 60 槽环形时间分桶

## 1. 项目目标

设计一个适用于 Go 高并发读多写少场景的 Cache 数据结构，使用双 `sync.Map` + 60 槽环形时间分桶（Circular Time Buckets）实现 TTL 过期数据的异步清理。

核心设计思想：

**业务数据访问与过期数据维护分离，以额外内存空间换取业务读取路径的性能。**

主要目标：

1. `m1` 专注业务数据存储，保证业务读取路径简单、高效。
2. 业务读取只访问 `m1`，不访问过期索引。
3. 不对 `m1` 执行全量 `Range()`。
4. `m2` 负责存储 KEY 对应的过期时间戳。
5. 使用固定 60 个时间 Bucket，每个 Bucket 对应 1 分钟。
6. 后台清理协程每分钟检查一个 Bucket。
7. 每 60 分钟完成一轮全部 Bucket 的检查。
8. 支持同一个 KEY 多次更新 TTL。
9. 支持业务读写与后台清理并发执行。
10. 允许过期数据在内存中短暂残留，降低清理频率和业务路径开销。

## 2. 整体架构

### 2.1 组成结构

整个 Cache 由三部分组成：

- m1：业务数据 Map。
- m2：60 个物理时间 Bucket，每个 Bucket 是一个 sync.Map。
- Cleanup Goroutine：负责每分钟检查一个 Bucket。

逻辑结构：

```text
                    Cache
                      |
          +-----------+-----------+
          |                       |
         m1                      m2
          |                       |
    sync.Map               60 Time Buckets
          |                       |
    key -> Entry          +-------+-------+
                          |       |       |
                        B[0]    B[1]   ... B[59]
                          |       |       |
                       sync.Map sync.Map sync.Map
                          |
                     key -> expireAt

                      |
                 Cleanup Goroutine
                      |
              每分钟检查一个 Bucket
                      |
               60 分钟完成一轮
```

### 2.2 核心职责

| 组件 | 职责 |
|---|---|
| m1 | 存储业务数据 |
| m2 | 存储过期时间索引 |
| Bucket | 按过期时间组织 KEY |
| Cleanup | 定时清理过期数据 |

严格约束：

- 业务 Load 不访问 m2。
- 业务 Load 不访问 Bucket。
- 业务 Load 不获取任何额外业务锁。
- Cleanup 不执行 m1.Range。
- Cleanup 只遍历当前需要检查的 Bucket。

## 3. 数据结构设计

### 3.1 Entry

m1 中不直接存储业务 Value，而是存储 Entry 指针。

```go
type Entry struct {
    Value     any
    ExpireAt  int64
    BucketSlot int
}
```

字段说明：

| 字段 | 说明 |
|---|---|
| Value | 业务数据 |
| ExpireAt | 绝对过期时间，UnixNano |
| BucketSlot | 对应的物理 Bucket 编号 |

Entry 采用不可变设计。

每次 Store 都创建新的 Entry，不允许原地修改已有 Entry 的过期时间。

Entry 指针本身可以作为版本标识，不需要额外 Version 字段。

### 3.2 Bucket

```go
type Bucket struct {
    Items sync.Map // key -> int64(expireAt)
}
```

m2 中只存储 KEY 和过期时间戳。

不需要在 m2 中重复保存完整业务数据。

### 3.3 Cache

```go
const BucketCount = 60

type Cache struct {
    m1 sync.Map // key -> *Entry

    buckets [BucketCount]Bucket

    // 只用于 Store、Delete、Cleanup 之间
    // 同一个 KEY 的操作串行化
    keyLocks [256]sync.Mutex

    bucketDuration time.Duration
}
```

说明：

- m1 和 m2 是完全独立的 sync.Map。
- buckets 是固定长度数组，不需要动态创建和删除 Bucket。
- keyLocks 是分片锁，仅保护同一个 KEY 的复合操作。
- Load 不需要使用 keyLocks。
- Bucket 本身不需要额外的全局锁。

分片锁数量可以配置，256 作为第一版默认值。

## 4. 时间分桶规则

### 4.1 基本配置

```go
BucketCount    = 60
BucketDuration = 1 * time.Minute
```

固定 60 个物理 Bucket。

每个 Bucket 对应一个分钟时间槽。

完整轮询周期：

```text
60 * 1 minute = 60 minutes
```

### 4.2 Bucket 编号计算

使用 Unix 时间计算绝对分钟编号。

```go
func minuteID(timestamp int64) int64 {
    return timestamp / int64(time.Minute)
}
```

物理 Bucket 编号：

```go
func bucketSlot(minuteID int64) int {
    return int(minuteID % BucketCount)
}
```

例如：

| 绝对分钟编号 | 物理 Bucket |
|---|---|
| 1000 | 40 |
| 1001 | 41 |
| 1002 | 42 |
| 1060 | 40 |
| 1061 | 41 |

同一个物理 Bucket 会每隔 60 分钟重复使用。

因此，物理 Bucket 编号不能直接代表过期时间，必须结合 Entry.ExpireAt 判断。

### 4.3 避免当前分钟的扫描竞争

需要注意一个边界问题：

假设 Cleanup 已经检查过当前分钟对应的 Bucket，此时业务协程才写入一个过期时间落在当前分钟的 KEY。

如果直接按照过期时间计算 Bucket，该 KEY 可能需要等待下一轮 60 分钟才会被检查。

因此，Store 时计算一个最早检查分钟：

```go
currentMinute := minuteID(time.Now().UnixNano())

expireMinute := minuteID(expireAt)

dueMinute := max(expireMinute, currentMinute+1)

slot := bucketSlot(dueMinute)
```

其中：

- `dueMinute` 是该 Entry 最早可以被检查的分钟。
- `ExpireAt` 仍然是真正的过期时间。
- 不能因为到达 dueMinute 就直接删除数据，仍然需要判断真实 ExpireAt。

这样可以避免新写入的数据因为当前 Bucket 已经扫描而错过本轮检查。

对于 TTL 大于 60 分钟的数据，物理 Bucket 会提前被检查一次或多次，但只要尚未过期，就保留索引，等待下一轮检查。

## 5. 核心 API 设计

### 5.1 Store

接口：

```go
Store(key, value any, ttl time.Duration)
```

执行流程：

1. 获取 KEY 对应的分片锁。
2. 计算绝对过期时间 ExpireAt。
3. 创建新的 Entry。
4. 计算 dueMinute 和 BucketSlot。
5. 更新 m1。
6. 将 KEY 和 ExpireAt 写入对应 Bucket。
7. 释放分片锁。

伪代码：

```go
func (c *Cache) Store(
    key, value any,
    ttl time.Duration,
) {
    lock := c.getKeyLock(key)
    lock.Lock()
    defer lock.Unlock()

    expireAt := time.Now().Add(ttl).UnixNano()

    currentMinute := minuteID(time.Now().UnixNano())
    expireMinute := minuteID(expireAt)

    dueMinute := max(expireMinute, currentMinute+1)
    slot := bucketSlot(dueMinute)

    entry := &Entry{
        Value:      value,
        ExpireAt:   expireAt,
        BucketSlot: slot,
    }

    // 先更新业务数据
    c.m1.Store(key, entry)

    // 再更新过期索引
    c.buckets[slot].Items.Store(key, expireAt)
}
```

注意：

- 同一个 KEY 的 Store 必须串行化。
- 每次 Store 创建新的 Entry。
- TTL 更新不需要主动删除旧 Bucket 中的索引。
- 旧索引由 Cleanup 自然回收。

TTL <= 0 的行为需要明确约定，建议直接执行 Delete，不写入过期索引。

### 5.2 Load

接口：

```go
Load(key any) (any, bool)
```

执行流程：

只访问 m1。

```go
func (c *Cache) Load(key any) (any, bool) {
    value, ok := c.m1.Load(key)
    if !ok {
        return nil, false
    }

    entry := value.(*Entry)

    return entry.Value, true
}
```

不访问 m2，不访问 Bucket，不使用分片锁。

第一版采用异步过期语义：

- KEY 到达 ExpireAt 后，不保证立即从 m1 消失。
- 后台 Cleanup 负责回收。
- 在清理完成前，Load 仍可能读取到已经超过 TTL 的数据。

如果业务需要严格 TTL 语义，后续可以在 Load 中增加 ExpireAt 判断，但这属于另一种读取策略。

### 5.3 Delete

接口：

```go
Delete(key any)
```

执行流程：

1. 获取 KEY 对应的分片锁。
2. 读取 m1 当前 Entry。
3. 删除 m1 中的业务数据。
4. 如果存在 Entry，使用 Entry.BucketSlot 定位当前索引。
5. 使用 CompareAndDelete 删除对应 Bucket 索引。
6. 释放锁。

旧 Bucket 中的历史索引不需要全部主动删除。

它们会在对应 Bucket 下一次被检查时发现已经失效并回收。

## 6. Cleanup 设计

### 6.1 调度方式

单独启动一个后台 Goroutine。

每分钟推进一个 Bucket。

```text
Minute 1  -> Bucket 0
Minute 2  -> Bucket 1
Minute 3  -> Bucket 2
...
Minute 60 -> Bucket 59
Minute 61 -> Bucket 0
```

实际 Bucket 编号根据 Unix 分钟计算，不依赖进程启动时间。

### 6.2 Cleanup 流程

每次 Cleanup：

1. 获取当前时间。
2. 确定当前应该检查的 Bucket。
3. Range 当前 Bucket。
4. 对每个 KEY 获取对应分片锁。
5. 重新读取当前 Bucket 索引，避免处理过期的 Range 快照。
6. 查询 m1 当前 Entry。
7. 判断当前 Entry 是否仍然属于该 Bucket。
8. 如果索引已经失效，删除旧索引。
9. 如果数据已经过期，删除 m1 和对应 m2 索引。
10. 如果数据尚未过期，保留索引。

### 6.3 核心判断逻辑

伪代码：

```go
func (c *Cache) cleanupBucket(slot int) {
    now := time.Now().UnixNano()

    bucket := &c.buckets[slot]

    bucket.Items.Range(func(key, value any) bool {
        lock := c.getKeyLock(key)
        lock.Lock()
        defer lock.Unlock()

        // 重新读取索引，防止 Range 期间发生更新
        currentExpire, ok := bucket.Items.Load(key)
        if !ok {
            return true
        }

        expireAt := currentExpire.(int64)

        // 确认索引仍然是本次 Range 看到的版本
        if expireAt != value.(int64) {
            return true
        }

        current, exists := c.m1.Load(key)

        if !exists {
            bucket.Items.CompareAndDelete(key, expireAt)
            return true
        }

        entry := current.(*Entry)

        // 当前 Entry 已经不属于这个 Bucket，
        // 说明这是 TTL 更新前留下的旧索引
        if entry.BucketSlot != slot ||
            entry.ExpireAt != expireAt {

            bucket.Items.CompareAndDelete(key, expireAt)
            return true
        }

        // 当前 Entry 已过期
        if now >= entry.ExpireAt {
            c.m1.CompareAndDelete(key, entry)
            bucket.Items.CompareAndDelete(key, expireAt)
        }

        return true
    })
}
```

重要说明：

分片锁保证 Store、Delete 和 Cleanup 对同一个 KEY 的复合操作串行化。

Entry 指针保证 m1 删除时只删除当前确认的 Entry。

m2 的过期时间戳校验保证只删除对应版本的索引。

### 6.4 旧索引回收机制

假设：

第一次写入：

```text
KEY = A
ExpireAt = 10:05
Bucket = 5
```

第二次更新 TTL：

```text
KEY = A
ExpireAt = 11:05
Bucket = 5（下一轮）
```

此时：

```text
m1:
A -> Entry(11:05)

Bucket 5（旧索引）:
A -> 10:05

Bucket 5（新一轮索引）:
A -> 11:05
```

当旧 Bucket 被检查时：

1. 读取旧索引 A -> 10:05。
2. 查询 m1 当前 Entry。
3. 发现当前 Entry.ExpireAt = 11:05。
4. 判断旧索引已经失效。
5. 删除旧索引。
6. 保留 m1 当前业务数据。

因此：

- TTL 更新不需要主动迁移旧索引。
- 允许临时存在重复索引。
- 旧索引在对应 Bucket 被检查时自然回收。
- 不会因为旧索引误删新数据。

如果新旧过期时间落在同一个物理 Bucket，新的 Store 会直接覆盖该 Bucket 中相同 KEY 的旧时间戳，不会产生额外记录。

## 7. 并发安全设计

### 7.1 同一个 KEY 并发 Store

使用 KEY 分片锁。

确保：

```text
m1.Store
    +
Bucket.Store
```

作为一个逻辑原子操作执行。

### 7.2 Store 与 Cleanup 并发

Store 和 Cleanup 使用相同的 KEY 分片锁。

避免出现：

- Cleanup 读取旧 Entry。
- Store 更新新 Entry。
- Cleanup 删除新数据。

### 7.3 不同 KEY 并发操作

不同 KEY 可能映射到同一分片锁，但不会影响数据正确性。

分片锁数量可以根据写入并发度进行 Benchmark 调整。

Load 完全不使用这些锁。

### 7.4 m1 和 m2 不要求实时一致

允许短暂存在：

- m1 有数据，m2 尚未写入。
- m2 有旧索引，m1 已经更新。
- m2 有旧索引，m1 已经删除。

这些情况由分片锁保护复合写操作，并由后台清理逐步回收。

最终要求是：

**m1 中的当前业务数据不会被旧索引误删，m2 中的失效索引能够最终回收。**

## 8. 定时任务与异常恢复

### 8.1 正常运行

每分钟执行一次：

```go
ticker := time.NewTicker(time.Minute)
```

根据当前绝对分钟计算 BucketSlot。

### 8.2 进程启动

建议启动时先对 60 个 Bucket 执行一轮完整扫描。

避免进程重启期间错过的过期索引继续残留。

随后进入每分钟检查一个 Bucket 的正常调度。

### 8.3 清理任务延迟

如果某次 Cleanup 耗时超过一分钟：

- 不允许启动多个 Cleanup 并发扫描相同 Bucket。
- 记录实际清理进度。
- 恢复后补扫遗漏的 Bucket。
- 如果停顿超过 60 分钟，至少确保 60 个物理 Bucket 都被扫描一次。

清理逻辑必须以真实 ExpireAt 判断是否删除，不能仅根据 Bucket 编号判断。

## 9. 性能设计要求

### 9.1 业务读取

业务读取路径：

```text
Load
  |
m1.Load
  |
返回 Value
```

不允许：

- m2.Load。
- Bucket 查询。
- 全局 Mutex。
- 额外的索引查询。

### 9.2 业务写入

业务写入路径：

```text
Store
  |
KEY 分片锁
  |
创建 Entry
  |
m1.Store
  |
Bucket.Store
  |
释放锁
```

写入操作相比单 Map 增加一次索引写入，这是空间换时间的主要成本。

### 9.3 后台清理

后台只 Range 一个 Bucket。

不允许 m1.Range。

建议第一版单 Goroutine 执行，避免多个清理任务竞争。

### 9.4 GC

重点关注：

- Entry 对象数量。
- 60 个 Bucket 中的索引数量。
- 失效索引回收速度。
- Map 扩容产生的内存开销。
- 长期运行后的 Heap 和 GC CPU。

## 10. 配置建议

第一版采用以下配置：

| 配置 | 默认值 |
|---|---|
| BucketCount | 60 |
| BucketDuration | 1 分钟 |
| FullCycle | 60 分钟 |
| CleanupConcurrency | 1 |
| KeyLockShards | 256 |
| Entry | 不可变指针 |
| m1 | sync.Map |
| m2 | 60 个 sync.Map |
| 删除方式 | CompareAndDelete |
| TTL 语义 | 异步过期 |

## 11. Benchmark 与验证

至少实现以下测试。

### 11.1 读取性能

比较：

1. 单 sync.Map Load。
2. 双 sync.Map + Bucket Load。

重点观察：

- ns/op。
- allocs/op。
- 并发吞吐量。
- P99 延迟。

预期：读取路径不应因为引入 m2 而产生额外开销。

### 11.2 写入性能

测试：

- 新 KEY Store。
- 同 KEY 重复 Store。
- TTL 更新后进入相同 Bucket。
- TTL 更新后进入不同 Bucket。

### 11.3 清理性能

分别测试：

- 10 万 KEY。
- 100 万 KEY。
- 300 万 KEY。
- 500 万 KEY。
- 1000 万 KEY。

重点观察：

- 单 Bucket Range 耗时。
- 每分钟清理耗时。
- 每次删除数量。
- CPU 峰值。
- GC CPU 占比。

### 11.4 并发安全

必须覆盖：

- Store 与 Cleanup 并发。
- Store 与 Delete 并发。
- 同 KEY 连续更新 TTL。
- 旧 Bucket 与新 Bucket 同时存在。
- TTL 超过 60 分钟。
- TTL 小于 1 分钟。
- Bucket 循环复用。
- Cleanup 执行期间写入同一 Bucket。
- 进程长时间暂停后恢复。

使用：

```bash
go test -race ./...
```

进行竞态检测。

## 12. 开发边界与后续优化

第一版不引入：

- Timing Wheel 多层时间轮。
- 动态 Bucket 数量。
- 多清理 Goroutine。
- 主动扫描 m1。
- 复杂的索引迁移机制。
- 额外的全局索引 Map。

先验证当前架构的性能和正确性。

后续根据 Benchmark 决定是否需要：

1. 增加 Bucket 内部分批清理。
2. 优化 KEY 分片锁。
3. 使用更大的 Bucket 数量。
4. 使用分层时间轮。
5. 调整清理调度策略。

## 13. 最终设计原则

整个方案遵循以下原则：

1. m1 是唯一业务数据读取入口。
2. m2 是独立的过期索引。
3. 业务读取不承担 TTL 索引成本。
4. 后台清理不遍历 m1。
5. 固定 60 个物理 Bucket，每分钟检查一个。
6. 每 60 分钟完成一轮全量 Bucket 检查。
7. 允许旧索引临时存在，不强制执行复杂迁移。
8. 通过当前 Entry 校验防止旧索引误删新数据。
9. 通过分片锁保证同一个 KEY 的 Store、Delete 和 Cleanup 操作安全。
10. 以实际 Benchmark 决定后续优化方向。

**核心目标：用独立的时间索引和后台异步清理机制，尽可能保护 m1 的高频读取性能。**




