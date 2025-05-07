package utils_json

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func JSONEncode(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func JSONDecode(v string, result interface{}) error {
	err := json.Unmarshal([]byte(v), &result)
	return err
}

func JSONDecodeV2(v interface{}, result interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &result)
	return err
}

func MapFilter(input map[string]int, max int) map[string]int {
	if len(input) <= max {
		return input
	}
	result := make(map[string]int, max)
	for k, v := range input {
		if k == "" {
			continue
		}
		if v > 0 {
			result[k] = v
		}
		max--
		if max <= 0 {
			break
		}
	}
	return result
}

func MapInt64Filter(input map[string]int64, max int) map[string]int64 {
	if len(input) <= max {
		return input
	}
	result := make(map[string]int64, max)
	for k, v := range input {
		if k == "" {
			continue
		}
		if v > 0 {
			result[k] = v
		}
		max--
		if max <= 0 {
			break
		}
	}
	return result
}

// ParseStruct proto的结构转成指定的 add EmitUnpopulated
func ParseStruct(input proto.Message, result interface{}) error {
	b, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(input)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, &result)
	if err != nil {
		return err
	}
	return nil
}

// StructToMap Convert structpb.Struct to map[string]interface{}
func StructToMap(s *structpb.Struct) map[string]interface{} {
	result := make(map[string]interface{})
	if s != nil {
		for k, v := range s.GetFields() {
			result[k] = ConvertValue(v)
		}
	}
	return result
}

// ConvertValue converts structpb.Value to native Go types
func ConvertValue(v *structpb.Value) interface{} {
	switch kind := v.Kind.(type) {
	case *structpb.Value_NullValue:
		return nil
	case *structpb.Value_NumberValue:
		return kind.NumberValue
	case *structpb.Value_StringValue:
		return kind.StringValue
	case *structpb.Value_BoolValue:
		return kind.BoolValue
	case *structpb.Value_StructValue:
		return StructToMap(kind.StructValue)
	case *structpb.Value_ListValue:
		list := make([]interface{}, len(kind.ListValue.Values))
		for i, lv := range kind.ListValue.Values {
			list[i] = ConvertValue(lv)
		}
		return list
	default:
		return nil
	}
}

func GetValueFromMap(m map[string]interface{}, key string) interface{} {
	if val, ok := m[key]; ok {
		return val
	}
	return ""
}

func MapToStructPb(m map[string]interface{}) (*structpb.Struct, error) {
	pbStruct, err := structpb.NewStruct(m)
	if err != nil {
		return nil, fmt.Errorf("failed to convert map to structpb.Struct: %v", err)
	}
	return pbStruct, nil
}

func ContainsAny(str string, substrings []string) bool {
	for _, substring := range substrings {
		if strings.Contains(str, substring) {
			return true
		}
	}
	return false
}

func MergeMaps(map1, map2 map[string]interface{}) map[string]interface{} {
	// 创建一个新的 map，用于存储合并后的结果
	result := make(map[string]interface{})

	// 将 map1 的所有键值对复制到 result 中
	for k, v := range map1 {
		result[k] = v
	}

	// 将 map2 的所有键值对复制到 result 中，如果有相同的键，会覆盖 map1 的值
	for k, v := range map2 {
		result[k] = v
	}

	return result
}

func ParseMapInterface(m interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	b, err := json.Marshal(m)
	if err == nil {
		err = json.Unmarshal(b, &result)
	}

	return result, err
}
