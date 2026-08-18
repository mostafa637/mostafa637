package storage

import (
	"database/sql/driver"
	"fmt"
)

func changePrefix(args []driver.Value) (driver.Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("storage: change_prefix expects 3 arguments, got %d", len(args))
	}
	input, ok := blobValue(args[0])
	if !ok {
		return nil, fmt.Errorf("storage: change_prefix input is not a blob")
	}
	start, ok := integerValue(args[1])
	if !ok || start < 0 || start > int64(len(input)) {
		return nil, fmt.Errorf("storage: change_prefix invalid prefix length %v", args[1])
	}
	replacement, ok := blobValue(args[2])
	if !ok {
		return nil, fmt.Errorf("storage: change_prefix replacement is not a blob")
	}

	out := make([]byte, 0, len(replacement)+len(input)-int(start))
	out = append(out, replacement...)
	out = append(out, input[int(start):]...)
	return out, nil
}

func blobValue(v driver.Value) ([]byte, bool) {
	switch x := v.(type) {
	case []byte:
		return x, true
	case string:
		return []byte(x), true
	default:
		return nil, false
	}
}

func integerValue(v driver.Value) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case int32:
		return int64(x), true
	case int:
		return int64(x), true
	case float64:
		return int64(x), true
	default:
		return 0, false
	}
}
