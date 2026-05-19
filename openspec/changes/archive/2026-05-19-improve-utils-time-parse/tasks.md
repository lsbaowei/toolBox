# Tasks — improve-utils-time-parse

## 1. 统一时间戳解析

- [x] 1.1 新增内部 `parseUnixFromString(str string) (time.Time, error)`，委托 `ParseUnixTimestamp`
- [x] 1.2 `ParseTimeUTC` 对整数字符串改用 `parseUnixFromString`，结果 `.UTC()`；删除或内联 `parseUnixTimestampV1`
- [x] 1.3 `ParseTime` 对整数字符串改用 `parseUnixFromString`，结果 `.In(defaultTZ)`；移除 `1e12` 分界逻辑

## 2. 布局与时区

- [x] 2.1 梳理 `layouts` 列表，确认无时区项与 `hasZoneInfo` 判断一致
- [x] 2.2 修正 `ParseTime` 对无时区布局应用 `defaultTZ` 的逻辑（不依赖 `Location().String() == "UTC"`）
- [x] 2.3 为 `ParseTimeUTC` / `ParseTime` 补充包级或函数注释，说明 UTC、defaultTZ、安全时间戳区间

## 3. 测试

- [x] 3.1 将 `TestParseTimeUTC` / `TestParseTime` 改为表驱动，断言期望时刻或 `wantErr`
- [x] 3.2 新增 `ParseUnixTimestamp` 边界用例（秒/毫秒分界、非法值）
- [x] 3.3 删除或修正无效用例（如 `"Mon Jan _2 15:04:05 2006"`）
- [x] 3.4 运行 `go test ./utils_time/...` 并修复失败

## 4. 收尾

- [x] 4.1 运行 `openspec validate improve-utils-time-parse`
- [x] 4.2 在 README 任务 1 下简要记录支持格式与 **BREAKING**（若有）说明
