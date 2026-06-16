# utils_cache_version

`utils_cache_version` 用于第三方版本变更时选择当前请求应使用的缓存版本。它通过渐进释放、确定性分桶和粘性灰度，避免缓存 key 瞬时全量切到新版本，从而保护第三方接口不被突增流量打满。

## 适用场景

业务依赖第三方接口，并在本地缓存第三方数据。缓存周期可能较长，例如 1-7 天；当第三方发布新版本时，业务希望更快刷新缓存，但不能让所有请求立刻打到第三方。

`utils_cache_version` 解决的是“本次请求应该拼接哪个版本到缓存 key”这个问题，不负责缓存读写，也不负责请求第三方接口。

## 安装与导入

```go
import "github.com/lsbaowei/toolBox/utils_cache_version"
```

## 核心类型

### `Config`

```go
type Config struct {
    ReleaseDuration time.Duration
    Now             func() time.Time
}
```

- `ReleaseDuration`：从稳定版本逐步释放到目标版本的窗口，必须大于 0。
- `Now`：可选时间源，便于测试或业务注入固定时间；为空时使用 `time.Now`。

### `SelectOptions`

```go
type SelectOptions struct {
    BusinessID string
    Version    string
    Now        time.Time
}
```

- `BusinessID`：稳定且分布足够均匀的业务标识，如用户 ID、租户 ID、资源 ID。
- `Version`：第三方当前返回的最新版本。
- `Now`：本次选择使用的当前时间；零值时使用 `Config.Now` 或 `time.Now`。

### `Result`

```go
type Result struct {
    Version       string
    StableVersion string
    TargetVersion string
    InRelease     bool
    Released      bool
    Progress      float64
}
```

- `Version`：调用方应拼接到缓存 key 中的版本。
- `StableVersion`：当前稳定版本。
- `TargetVersion`：正在释放的目标版本，没有活跃释放时为空。
- `InRelease`：是否处于释放阶段。
- `Released`：当前业务标识是否命中目标版本。
- `Progress`：释放进度，范围为 `[0, 1]`。

## 基本用法

```go
mgr := utils_cache_version.New(utils_cache_version.Config{
    ReleaseDuration: 3 * time.Hour,
})

got, err := mgr.SelectVersion(utils_cache_version.SelectOptions{
    BusinessID: "user-123",
    Version:    "v1.1.4",
})
if err != nil {
    // handle error
}

cacheKey := "third-party:data:" + got.Version
_ = cacheKey
```

## 行为说明

- 首次观察到的版本会直接成为稳定版本，不启动灰度释放。
- 当第三方返回不同于稳定版本的新版本时，开启释放阶段。
- 释放阶段内，按照 `BusinessID + TargetVersion` 的确定性分桶判断是否释放到目标版本。
- 已经释放到目标版本的 `BusinessID` 会保持粘性，后续请求不会回退到旧版本。
- 释放期间如果第三方继续返回更新版本，活跃目标会更新到最新版本，中间快速版本不单独排队。
- 释放窗口完成后，目标版本成为新的稳定版本，并清理该阶段粘性状态。
- 稳定版本高频命中路径使用读锁快速返回，不需要 `BusinessID`。

## 错误

- `ErrInvalidReleaseDuration`：`ReleaseDuration <= 0`。
- `ErrEmptyVersion`：第三方版本为空。
- `ErrEmptyBusinessID`：已有稳定版本后，需要进入释放计算但业务标识为空。

## 测试时间

可以通过 `SelectOptions.Now` 或 `Config.Now` 控制时间，便于编写稳定测试。

```go
now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
mgr := utils_cache_version.New(utils_cache_version.Config{
    ReleaseDuration: 3 * time.Hour,
    Now: func() time.Time {
        return now
    },
})

_, _ = mgr.SelectVersion(utils_cache_version.SelectOptions{Version: "v1"})
now = now.Add(time.Hour)
```

## 并发与限制

- `Manager` 是并发安全的。
- 默认状态只保存在当前进程内。
- 多实例场景需要业务侧接入共享状态，或保证同一业务标识路由到同一实例。
- `BusinessID` 的分布会影响释放均匀度，应避免使用低基数或高度倾斜的标识。
