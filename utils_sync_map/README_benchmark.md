# 与 sync.Map 的性能对比

对照实现是标准库 `sync.Map`。`durationring` 和 `minutering` 的 `Load` 只读业务表 `m1`，下面的数字用来确认这条路径没有把读取变慢。

测量环境：Apple M4，`go test -benchmem -benchtime=2s`。预填充 10000 个 key。`-cpu` 用来提高 `RunParallel` 的并行度。数字会随机器和 Go 版本变化，看的是同一轮里和 `sync.Map` 的差距，而不是绝对值。

## 结论

1. **并发读取和 `sync.Map` 持平。** 并行度从 1 拉到 32，`Load` 的差距不扩大。32 并行时三者都在 15 ns/op 附近，差值小于 1 ns。
2. **后台清理不拖慢 `Load`。** 清理协程同时运行时，读取耗时与只跑 `Load` 相同。清理不走业务读路径，也不给 `Load` 加锁。
3. **写入更慢，这是设计上的代价。** `Store` 要拿分片锁，写业务表，再写一条过期索引。单协程大约是 `sync.Map` 的 2～3 倍，并发写入大约 2 倍。`minutering` 的索引只存 `int64`，写入略快于 `durationring`。
4. **读多写少时，差距来自那一小部分写。** 80% `Load` + 20% `Store` 的混合负载，差距稳定在约 37%，不随并行度从 4 升到 32 而变大。读的比例越高，整体越接近 `sync.Map`。

## 并发读取

单位 ns/op，越低越好。

| 并行度 | `sync.Map` | `durationring` | `minutering` |
| --- | ---: | ---: | ---: |
| 1 | 69 | 71 | 71 |
| 4 | 20 | 20 | 20 |
| 8 | 17 | 17 | 17 |
| 16 | 16 | 15 | 16 |
| 32 | 15 | 15 | 15 |

单协程大约慢 2～4 ns，来自多一次 `*Entry` 解引用。并发之后这个差值被摊平。

## 读取时清理协程同时运行

| 并行度 | `durationring` | `minutering` |
| --- | ---: | ---: |
| 1 | 71 | 72 |
| 8 | 17 | 17 |
| 16 | 15 | 15 |
| 32 | 16 | 16 |

和上一节只跑 `Load` 的数字一致。

## 写入

默认并行度（本机 `-cpu` 未指定时为 10）下的单次操作：

| | `sync.Map` | `durationring` | `minutering` |
| --- | ---: | ---: | ---: |
| `Store` 单协程 | 114 ns/op | 323 ns/op | 228 ns/op |
| `Store` 并发 | 54 ns/op | 129 ns/op | 111 ns/op |

## 80% 读 + 20% 写

`minutering` 与 `sync.Map` 的混合负载。差距不随并行度上升。

| 并行度 | `sync.Map` | `minutering` | 差距 |
| --- | ---: | ---: | ---: |
| 1 | 88 ns/op | 114 ns/op | +30% |
| 4 | 25 ns/op | 35 ns/op | +40% |
| 8 | 25 ns/op | 37 ns/op | +47% |
| 16 | 23 ns/op | 31 ns/op | +38% |
| 32 | 22 ns/op | 31 ns/op | +37% |

## 怎么复现

```shell
go test -run='^$' -bench=. -benchmem -benchtime=2s ./utils_sync_map/durationring/
go test -run='^$' -bench=. -benchmem -benchtime=2s ./utils_sync_map/minutering/
go test -run='^$' -bench='LoadParallel|LoadWithCleanup|MixedParallel' -benchmem -benchtime=2s -cpu=1,4,8,16,32 ./utils_sync_map/durationring/
go test -run='^$' -bench='LoadParallel|LoadWithCleanup|MixedParallel' -benchmem -benchtime=2s -cpu=1,4,8,16,32 ./utils_sync_map/minutering/
```
