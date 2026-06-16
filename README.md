# Toolkit for golang
    简单好用，按照功能分文件夹（包）存放

## 任务1
    基本功能：`utils_time` 将时间字符串（Unix 秒/毫秒、常见 RFC/自定义布局）解析为 `time.Time`。

    **支持形态**
    - Unix：纯十进制整数；`0 < 秒 < 1e10`，`1e10 <= 毫秒 < 1e13`（与 `ParseUnixTimestamp` 一致）
    - 布局：`2006-01-02`、`2006-01-02 15:04:05`、RFC3339/1123/822、ANSIC、UnixDate、RubyDate 等（见 `utils_time/v1.go` 中 `layouts`）

    **入口**
    - `ParseTimeUTC`：无时区布局按 UTC 墙钟；含 offset 的布局保留原偏移
    - `ParseTime(str, defaultTZ)`：无时区布局按 `defaultTZ` 墙钟（`nil` 时用 `Local`）
    - `ParseUnixTimestamp`：仅解析整数时间戳

    **BREAKING（相对旧版）**
    - 时间戳 `0`、负数、超出安全区间、非纯数字串不再被当作 Unix 成功解析
    - `ParseTime` / `ParseTimeUTC` 的 Unix 路径统一走 `ParseUnixTimestamp`（不再使用 `1e12` 或无前缀零的宽松逻辑）

## 任务2：时间对象 `DateTime`

    不可变时间包装，方法返回新 `*DateTime`。

    **构造**
    - `New(nil)` → 当前时间
    - `New(&t)` → 指定基础时间

    **基础方法**
    - `Time()` → `time.Time`
    - `Format` / `FormatRFC3339` / `FormatDate` / `FormatDateTime`
    - `Add` / `AddDate`

    **边界（沿用基础时间的 Location）**
    - `StartOfDay` / `EndOfDay`（结束为 `23:59:59.999999999`）
    - `StartOfWeek` / `EndOfWeek`（周一至周日，结束 `23:59:59`）
    - `StartOfMonth` / `EndOfMonth`（结束 `23:59:59`）

    **示例**

    ```go
    loc, _ := time.LoadLocation("Asia/Shanghai")
    t, _ := ParseTime("2024-06-15 14:30:00", loc)
    d := New(&t)
    _ = d.StartOfDay().FormatDate()           // 2024-06-15
    _ = d.EndOfDay().FormatDateTime()         // 2024-06-15 23:59:59
    _ = d.StartOfWeek().AddDate(0, 0, 7)     // 链式：下周同一时刻
    ```

## 任务2.1：`DateTime` 辅助方法

    - `RemainingSeconds(other *time.Time) int64` — `基础时间.Unix() - other.Unix()`；`other == nil` 时用当前时间；可为负
    - `Random(max int64) int64` — 以当前毫秒时间戳为种子的伪随机数，`[0, max)`；`max <= 0` 返回 `0`

    ```go
    d := New(&expireAt)
    sec := d.RemainingSeconds(nil)  // 距现在剩余秒
    n := d.Random(100)              // [0, 100)
    ```

## 任务3：代码审查与加固（已完成）

    全库审查与修复见 [CHANGELOG.md](CHANGELOG.md)。摘要：修复 `JSONDecode`、随机数上界、`OnceFullWrite` panic、`ExecCmd` 上下文取消等；demo 迁至 `cmd/csvdemo`；补充多包单元测试。

## 任务4：随机数优化（已完成）

    **伪随机**（复用 `*rand.Rand`，适合抖动、抽样、非安全场景）

    ```go
    d := utils_time.New(nil)
    n := d.Random(100)                    // DateTime，毫秒因子播种

    ru := utils_random.New()
    x := ru.Intn(100)                     // 独立实例，线程安全

    y := utils_random.IntV2()             // 包级共享源
    ```

    **安全随机**（`crypto/rand`，适合 token、密钥、不可预测场景）

    ```go
    v, err := utils_random.SecureIntn(100) // [0, 100)
    w, err := utils_random.SecureInt64()  // [0, 2^63)
  // 或 utils_random.IntWithSafety()
    ```

    详见 [CHANGELOG.md](CHANGELOG.md) 中 `optimize-rand-reuse` 条目。


## 任务5：utils_time/v1.go（已完成）

    `parseWithLayouts` 支持以下常见 ISO 类时间字符串：

    - `2025-11-19T11:33:19.920584349+08:00` — `RFC3339Nano`
    - `2026-05-18T15:29:04.527+08:00` — `RFC3339Nano`
    - `2025-01-02T15:04:05+08:00` — `RFC3339`
    - `2025-01-02T15:04:05` — `2006-01-02T15:04:05`（墙钟，无时区）

## 任务6：缓存版本渐进释放

    `utils_cache_version` 用于第三方版本变更时选择本次请求应使用的缓存版本，避免缓存 key 立刻全量切到新版本导致第三方接口流量突增。

    **入口**
    - `New(Config)`：创建单进程内的版本释放管理器
    - `SelectVersion(SelectOptions)`：传入业务标识、第三方当前版本和当前时间，返回应使用的缓存版本

    **行为**
    - 首次观察到的版本会直接成为稳定版本
    - 后续发现新版本后，在 `ReleaseDuration` 内按业务标识确定性分桶逐步释放
    - 已释放到目标版本的业务标识会保持粘性，后续不会回退到旧版本
    - 释放期间如果第三方继续返回更新版本，活跃目标会更新到最新版本，中间快速版本不单独排队

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

    `BusinessID` 应选择稳定且分布足够均匀的业务标识，如用户 ID、租户 ID 或资源 ID。默认 `Manager` 只维护当前进程内状态，多实例场景需要业务侧接入共享状态或保证同一业务标识路由到同一实例。

