# 环形时间分桶缓存

`utils_sync_map` 提供两种面向读多写少场景的进程内缓存。两者都用双 `sync.Map` 把业务数据和过期索引分开：业务 `Load` 只访问数据表，后台协程按时间桶回收过期数据，避免对业务表做全量 `Range`。

| 包 | 类型 | 适用场景 |
| --- | --- | --- |
| [`durationring`](durationring) | `DurationRing` | Bucket 数量固定，时间粒度可配置；清理周期较长 |
| [`minutering`](minutering) | `MinuteRing` | 固定 60 个 1 分钟槽，每分钟检查一个 Bucket |

设计说明见同目录的 `README_task_1.md` 与 `README_task_2.md`。

## 共同语义

- `Store(key, value, ttl)` 写入数据。`ttl <= 0` 时等价于 `Delete`。
- `Load` 只读业务表，不查过期索引，也不加分片锁。
- 过期是异步的：到达过期时间后，在后台清理完成前，`Load` 仍可能读到该数据。
- 每次 `Store` 创建新的不可变 `Entry`。同一 key 更新 TTL 时不主动迁移旧索引，旧索引由清理回收，不会误删新数据。
- 同一 key 的 `Store`、`Delete` 与清理通过分片锁串行化；`Load` 不参与这把锁。
- `key` 必须是可比较类型，要求与 `sync.Map` 相同。
- `New` 会启动后台清理协程，使用结束后调用 `Close`。`Close` 可安全多次调用。

## durationring

固定 Bucket 数量，用 `BucketDuration` 调整单个 Bucket 覆盖的时间范围。索引保存完整的绝对时间槽编号，TTL 长于整圈覆盖窗口时，也不会因为物理槽复用而被提前删除。

默认配置：

| 配置 | 默认值 |
| --- | --- |
| `BucketCount` | 60 |
| `BucketDuration` | `6 * time.Minute` |
| `CleanupInterval` | `2 * time.Hour` |
| `KeyLockShards` | 256 |

启动时先完整扫描一轮，之后每隔 `CleanupInterval` 再扫描全部 Bucket。零值字段在 `New` 时填入默认值。

```go
package main

import (
	"fmt"
	"time"

	"github.com/lsbaowei/toolBox/utils_sync_map/durationring"
)

func main() {
	cache := durationring.New(durationring.Config{
		BucketCount:     60,
		BucketDuration:  6 * time.Minute,
		CleanupInterval: 2 * time.Hour,
	})
	defer cache.Close()

	cache.Store("user:1", "alice", 30*time.Minute)

	if value, ok := cache.Load("user:1"); ok {
		fmt.Println(value)
	}

	cache.Delete("user:1")
}
```

## minutering

固定 60 个物理 Bucket，每个对应 1 分钟。后台按绝对分钟推进：每分钟检查一个 Bucket，约 60 分钟走完一轮。清理若延迟，会补扫遗漏的槽；停顿超过一整轮时，至少保证 60 个物理槽都被扫一次。同一时刻只跑一个清理任务。

写入时，最早可检查分钟不早于下一分钟，避免数据落进当前分钟、而该槽本轮已经扫过。真正是否删除仍看 `ExpireAt`，到达检查槽并不等于过期。

默认配置：

| 配置 | 默认值 |
| --- | --- |
| `BucketCount` | 60（固定） |
| `BucketDuration` | `time.Minute`（固定） |
| `CleanupTick` | `time.Minute` |
| `KeyLockShards` | 256 |

进程启动时默认先把 60 个 Bucket 各扫一遍。

```go
package main

import (
	"fmt"
	"time"

	"github.com/lsbaowei/toolBox/utils_sync_map/minutering"
)

func main() {
	cache := minutering.New(minutering.Config{})
	defer cache.Close()

	cache.Store("user:1", "alice", 30*time.Minute)

	if value, ok := cache.Load("user:1"); ok {
		fmt.Println(value)
	}

	cache.Delete("user:1")
}
```

## 不一致之处与适用场景

业务读写路径一致：`Load` 只读 `m1`，`Store` / `Delete` 持分片锁同时改业务表和过期索引。下面只列清理路径上真正不同的地方。

### 索引里存什么

| | `durationring` | `minutering` |
| --- | --- | --- |
| Bucket 值 | `indexMeta{expireAt, absoluteSlot}` | 只有 `expireAt`（`int64`） |
| 旧索引判定 | 物理槽、绝对槽、过期时间三者都对上才算当前索引 | 物理槽和过期时间对上即算当前索引 |

`durationring` 多存一份绝对时间槽，用来区分「同一物理槽上的不同轮次」。`minutering` 索引更轻，靠「最早检查分钟不早于下一分钟」加上真实 `ExpireAt` 避免漏检和误删。

### 清理时多出来的判断

`durationring` 每次会顺序扫完全部 Bucket，但索引的 `ExpireAt` 还没到时，不读也不写 `m1`。未到期数据留在业务表里，等它真正过期后的下一轮清理再核对。这样全量扫分片不会变成对 `m1` 的全量读取。

`minutering` 每次只扫一个槽，扫到的 key 都会读一次 `m1` 来判断是不是旧索引。是否删除仍只看 `now >= ExpireAt`。两边都不会因为槽被扫到就删掉还没到期的数据。

### 调度方式

| | `durationring` | `minutering` |
| --- | --- | --- |
| 每次清理 | 同一个协程顺序扫全部 Bucket，不并行 | 按绝对分钟推进一个 Bucket |
| 对 `m1` 的访问 | 只处理索引已到期的 key；每删 64 个就让出调度 | 扫到的 key 都会读 `m1` |
| 默认节奏 | `CleanupInterval` = 2 小时 | `CleanupTick` = 1 分钟，约 60 分钟走完一轮 |
| 时间粒度 | `BucketDuration` 可配，默认 6 分钟 | 固定 1 分钟，`BucketCount` 固定 60 |
| 清理延迟 | 过期数据最多再留一个清理周期（默认约 2 小时） | 过期数据通常在一轮内回收（约 1 小时） |
| 任务落后 | 下次触发再全量扫一遍 | 按 `lastCleanedMinute` 补扫遗漏槽；停顿超过一整轮时，至少把 60 个物理槽各扫一次 |
| 重入 | 只有一个清理协程，Bucket 之间不会并发 | `cleaning` 标志保证同一时刻只有一个清理任务 |

### 适用场景

**用 `durationring`：**

- 读多写少，能接受过期数据在内存里多留 1～3 小时。
- 希望清理尽量少打扰业务读：默认 2 小时才全量扫一次，而且是单协程逐个 Bucket 扫。未到期的索引不碰 `m1`，到期删除也按批让出调度，避免并行清理把 `Load` 拖慢。
- TTL 经常长于一整圈覆盖窗口（默认 60 × 6 分钟 = 6 小时）。绝对时间槽能认出「这个物理槽属于更后面的一轮」，长 TTL 不会在槽复用时被提前删掉。
- 需要按业务调整粒度：把 `BucketDuration` 调大，同一 key 更新 TTL 时跨槽迁移更少；调小则单 Bucket 里的 key 更少，单次清理更轻。

**用 `minutering`：**

- 希望过期数据大约在 1 小时内被收掉，不能接受按小时级残留。
- 写入的 TTL 大多在几分钟到 1 小时，1 分钟一槽、每分钟只 `Range` 一个 Bucket，单次清理的数据量更小、更平稳。
- 清理协程可能被 GC 或调度拖慢。补扫会把漏掉的分钟槽补上，长时间停顿后也会把 60 个物理槽至少扫一遍。
- 不需要配置时间粒度，固定 60 × 1 分钟即可。

两种实现都不提供条数或内存上限。只写入、长期不让清理跑完时，进程内存会持续增长。清理越惰性，过期数据占内存的时间越长。

## 测试

```shell
go test -race ./utils_sync_map/...
```
