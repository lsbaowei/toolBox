# utils_random

`utils_random` 提供伪随机和密码学安全随机工具。伪随机适合抽样、抖动、测试数据等非安全场景；安全随机适合 token、密钥片段、验证码种子等不可预测场景。

## 功能概览

- `RandUtil`：独立随机源，内部加锁，可并发使用。
- 包级伪随机函数：兼容旧调用方式。
- 可指定 seed 的确定性随机数，便于测试。
- `crypto/rand` 安全随机整数。

## 安装与导入

```go
import "github.com/lsbaowei/toolBox/utils_random"
```

## 伪随机

推荐使用 `RandUtil` 创建独立随机源：

```go
ru := utils_random.New()

n := ru.Intn(100)     // [0, 100)
f := ru.Float64()     // [0.0, 1.0)
s := ru.String(16)    // 随机字母数字字符串

_, _, _ = n, f, s
```

测试或调试时可以使用固定 seed：

```go
ru := utils_random.NewWithSeed(42)
first := ru.Intn(100)
second := ru.Intn(100)

_, _ = first, second
```

包级函数：

- `IntV2() int`：复用包内伪随机源。
- `IntWithT(seed int64) int`：使用固定 seed 生成一个随机数。
- `Int() int`：旧接口，已废弃。

## 安全随机

`SecureIntn(max int64) (int64, error)` 使用 `crypto/rand` 返回 `[0, max)` 范围内的安全随机整数。`max <= 0` 时返回 `0, nil`。

```go
n, err := utils_random.SecureIntn(100)
if err != nil {
    // handle error
}
_ = n
```

`SecureInt64() (int64, error)` 返回 `[0, 2^63)` 范围内的安全随机整数。

```go
n, err := utils_random.SecureInt64()
if err != nil {
    // handle error
}
_ = n
```

`IntWithSafety()` 是 `SecureInt64` 的兼容入口。

## 选择建议

- 普通抽样、负载抖动、测试数据：使用 `RandUtil` 或 `IntV2`。
- 需要可重复结果的测试：使用 `NewWithSeed` 或 `IntWithT`。
- token、验证码、安全敏感随机数：使用 `SecureIntn` 或 `SecureInt64`。

## 注意事项

- `math/rand` 不适合安全场景。
- `RandUtil.String` 生成的是字母数字字符串，不保证密码学安全。
- `Int()` 保留旧行为，每次调用会 seed，不建议新代码使用。
