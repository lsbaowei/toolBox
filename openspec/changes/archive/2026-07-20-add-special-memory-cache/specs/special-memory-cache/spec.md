## ADDED Requirements

### Requirement: 缓存零值可用且全局共享
系统 SHALL 提供零值即可调用的缓存类型，并 SHALL 让该类型的所有实例使用同一个进程级并发安全存储。

#### Scenario: 通过零值实例读写
- **WHEN** 调用方声明一个未显式初始化的缓存变量并写入条目
- **THEN** 同一变量能够读取到该条目

#### Scenario: 不同实例共享条目
- **WHEN** 调用方通过一个缓存实例写入条目，再通过另一个实例读取相同 key
- **THEN** 第二个实例能够读取到相同条目

### Requirement: 条目使用连续字节布局
系统 SHALL 将每个缓存条目保存为单个 `[]byte`，其中前 8 字节以固定字节序编码 `int64` Unix 秒级过期时间，其余字节保存业务数据。

#### Scenario: 编码带过期时间的数据
- **WHEN** 调用方写入业务数据和正数过期时间戳
- **THEN** 存储条目的前 8 字节可还原为该时间戳，剩余字节与业务数据一致

#### Scenario: 保存空业务数据
- **WHEN** 调用方写入空 `[]byte`
- **THEN** 系统仍保存包含 8 字节时间戳头部的有效条目

### Requirement: 写入拥有独立数据
`Set` SHALL 接收 key、业务数据和绝对过期时间，且 SHALL 在写入时取得业务数据的独立所有权，避免调用方后续修改输入切片影响缓存内容。系统 SHALL 将 `expireAt == 0` 解释为永不过期，并 SHALL 拒绝负数过期时间。

#### Scenario: 输入切片在写入后被修改
- **WHEN** 调用方成功写入数据后修改原始输入切片
- **THEN** 缓存中的业务数据保持写入时的内容

#### Scenario: 写入永不过期条目
- **WHEN** 调用方使用过期时间 `0` 写入条目
- **THEN** 该条目不会因时间推进而过期

#### Scenario: 拒绝负数过期时间
- **WHEN** 调用方使用负数过期时间写入条目
- **THEN** `Set` 返回错误且存储内容不发生变化

### Requirement: 读取返回拆分后的有效条目
`Get` SHALL 在条目有效时返回原始过期时间、业务数据和命中状态。返回的业务数据 SHALL 是存储条目去除 8 字节头部后的只读视图，不产生额外数据复制。

#### Scenario: 读取未过期条目
- **WHEN** 当前 Unix 时间早于条目的正数过期时间
- **THEN** `Get` 返回写入时的过期时间、业务数据以及命中状态

#### Scenario: 读取永不过期条目
- **WHEN** 条目的过期时间为 `0`
- **THEN** `Get` 返回该条目以及命中状态

#### Scenario: 读取不存在的 key
- **WHEN** 存储中不存在指定 key
- **THEN** `Get` 返回未命中

### Requirement: 过期条目不作为命中返回
当正数过期时间小于或等于当前 Unix 秒时，系统 SHALL 将条目视为过期。`Get` SHALL 返回该条目的过期时间和业务数据并返回未命中状态，但 SHALL NOT 自动删除该条目；过期数据由调用方主动管理。

#### Scenario: 到达过期边界
- **WHEN** 当前 Unix 时间等于条目的过期时间
- **THEN** `Get` 返回原过期时间和业务数据，同时返回未命中

#### Scenario: 过期读取保留数据
- **WHEN** `Get` 读取到过期条目并返回未命中
- **THEN** 该条目仍保留在底层存储中，直到调用方显式删除或覆盖

### Requirement: 支持显式删除和原子取出
系统 SHALL 提供按 key 删除条目的操作，并 SHALL 提供返回现有有效条目后将其删除的 `LoadAndDelete` 操作。对不存在或已过期条目执行 `LoadAndDelete` SHALL 返回未命中。

#### Scenario: 显式删除条目
- **WHEN** 调用方删除一个已存在的 key 后再次读取
- **THEN** 系统返回未命中

#### Scenario: 取出并删除有效条目
- **WHEN** 调用方对有效条目执行 `LoadAndDelete`
- **THEN** 调用返回其过期时间和业务数据，后续读取返回未命中

### Requirement: 遍历仅暴露有效条目
`Range` SHALL 遍历调用期间可观察到的有效条目，并向回调提供 key、过期时间和业务数据。遍历 SHALL 继承 `sync.Map.Range` 的弱一致性和提前停止语义，且 SHALL NOT 将过期条目传给回调。

#### Scenario: 遍历包含有效和过期条目
- **WHEN** 存储中同时存在有效条目和过期条目
- **THEN** 回调仅收到有效条目

#### Scenario: 回调要求停止
- **WHEN** 遍历回调返回 `false`
- **THEN** `Range` 停止继续调用该回调

### Requirement: 支持按需清理过期条目
系统 SHALL 提供 `DeleteExpired` 操作，接收用于判定的 Unix 秒时间，删除遍历时观察到的已过期条目并返回发起删除的数量。该操作 SHALL NOT 删除遍历时观察到的永不过期条目，但 SHALL 采用 `sync.Map.Range` 的弱一致性语义，不保证保留清理期间相同 key 的并发更新。

#### Scenario: 批量清理
- **WHEN** 调用方执行 `DeleteExpired` 且存储中存在多个已过期条目
- **THEN** 系统删除这些条目并返回实际删除数量

#### Scenario: 保留有效和永不过期条目
- **WHEN** 调用方执行 `DeleteExpired`
- **THEN** 过期时间晚于判定时间的条目及过期时间为 `0` 的条目仍可读取

#### Scenario: 清理期间并发覆盖
- **WHEN** `DeleteExpired` 观察到过期条目后，相同 key 被并发写入新条目
- **THEN** 系统不保证该并发写入的新条目在本次清理后仍然存在

### Requirement: 行为和分配可验证
实现 SHALL 提供单元测试覆盖并发存取、时间边界、全局共享、删除和遍历行为，并 SHALL 提供 benchmark 观察 `Set` 与 `Get` 的耗时和内存分配。

#### Scenario: 执行自动化验证
- **WHEN** 在支持 Go 1.19 的环境中运行该包测试
- **THEN** 单元测试通过且 benchmark 可独立执行
