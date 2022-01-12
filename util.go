package generator

import (
	"go/types"
	"strings"
	"unicode"
)

func getBasicTypeName(k types.BasicKind) string {
	switch k {
	case types.Bool:
		return "bool"
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
		return "int"
	case types.Float32, types.Float64:
		return "double"
	case types.String:
		return "String"
	default:
		return "dynamic" // Unsupported type
	}
}

func convertEnumKey(key string) string {
	if len(key) == 0 {
		return ""
	}

	builder := strings.Builder{}
	builder.WriteRune(unicode.ToLower(rune(key[0])))
	builder.WriteString(key[1:])

	return builder.String()
}

// SplitPackegeStruct - package.structを分割
func SplitPackegeStruct(s string) (string, string) {
	idx := strings.LastIndex(s, ".")

	return s[:idx], s[idx+1:]
}
