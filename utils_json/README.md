# utils_json

`utils_json` 提供 JSON 编解码、map 过滤、protobuf 结构转换和常用 map/string 辅助函数，适合接口数据转换和轻量结构化数据处理。

## 功能概览

- JSON 序列化与反序列化。
- 任意对象通过 JSON 中转转换为目标结构。
- `map[string]int` / `map[string]int64` 过滤。
- `proto.Message`、`structpb.Struct`、`structpb.Value` 与 Go 原生类型转换。
- map 合并、字符串包含判断、map 字段读取。

## 安装与导入

```go
import "github.com/lsbaowei/toolBox/utils_json"
```

## JSON 编解码

`JSONEncode(v interface{}) string` 序列化失败时返回空字符串，适合兼容旧调用；新代码更推荐 `JSONEncodeE`。

```go
s, err := utils_json.JSONEncodeE(map[string]interface{}{
    "name": "alice",
    "age":  18,
})
if err != nil {
    // handle error
}
_ = s
```

`JSONDecode(v string, result interface{}) error` 语义同 `encoding/json.Unmarshal`，`result` 必须是指针。

```go
var user struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

err := utils_json.JSONDecode(`{"name":"alice","age":18}`, &user)
if err != nil {
    // handle error
}
```

`JSONDecodeV2(v interface{}, result interface{}) error` 会先把 `v` marshal 成 JSON，再 unmarshal 到目标对象，适合 map/struct 之间做轻量转换。

## Map 过滤

`MapFilter(input map[string]int, max int) map[string]int` 和 `MapInt64Filter(input map[string]int64, max int) map[string]int64` 用于限制 map 大小。

当 `len(input) <= max` 时，会返回一份浅拷贝。超过 `max` 时，会跳过空 key，并优先保留值大于 0 的条目，直到达到 `max`。

```go
filtered := utils_json.MapFilter(map[string]int{
    "a": 1,
    "b": 2,
}, 1)
_ = filtered
```

## protobuf 转换

`ParseStruct(input proto.Message, result interface{}) error` 使用 `protojson.MarshalOptions{EmitUnpopulated: true}` 将 protobuf message 转为 JSON 后再写入目标结构。

```go
err := utils_json.ParseStruct(pbMsg, &dst)
if err != nil {
    // handle error
}
```

`StructToMap(s *structpb.Struct) map[string]interface{}` 会把 `structpb.Struct` 转成原生 map。

`MapToStructPb(m map[string]interface{}) (*structpb.Struct, error)` 会把原生 map 转成 `structpb.Struct`。

## 常用辅助函数

- `GetValueFromMap(m map[string]interface{}, key string) interface{}`：读取 map 字段，不存在时返回空字符串。
- `ContainsAny(str string, substrings []string) bool`：判断字符串是否包含任意子串。
- `MergeMaps(map1, map2 map[string]interface{}) map[string]interface{}`：合并两个 map，`map2` 同名字段覆盖 `map1`。
- `ParseMapInterface(m interface{}) (map[string]interface{}, error)`：通过 JSON 中转解析为 `map[string]interface{}`。

## 注意事项

- `JSONEncode` 会吞掉错误并返回空字符串；需要错误信息时请用 `JSONEncodeE`。
- `JSONDecodeV2`、`ParseMapInterface` 依赖 JSON marshal/unmarshal，字段类型会遵循 JSON 转换规则。
- `MergeMaps` 是浅合并，不会递归合并嵌套 map。
