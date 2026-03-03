package query

import (
	"fmt"
	"sort"
	"strconv"
)

// SortItems sorts items based on the ORDER BY options.
func SortItems(items []map[string]interface{}, queryOpts *QueryOptions) ([]map[string]interface{}, error) {
	orderOpts := queryOpts.OrderBy
	selectOpts := queryOpts.Select

	if len(orderOpts) == 0 {
		return items, nil
	}

	result := make([]map[string]interface{}, len(items))
	copy(result, items)

	sort.SliceStable(result, func(i, j int) bool {
		for _, orderOpt := range orderOpts {
			name := orderOpt.Field.Alias
			valueI, err := GetValue(result[i], name, selectOpts)
			if err != nil {
				continue
			}
			valueJ, err := GetValue(result[j], name, selectOpts)
			if err != nil {
				continue
			}

			valueI = convertStringToNumeric(valueI)
			valueJ = convertStringToNumeric(valueJ)

			cmp := compareValues(valueI, valueJ)
			if cmp == 0 {
				continue
			}

			if orderOpt.Descending {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})

	return result, nil
}

func convertStringToNumeric(value interface{}) interface{} {
	if strValue, ok := value.(string); ok {
		if i, err := strconv.ParseInt(strValue, 10, 64); err == nil {
			return int(i)
		}
		if f, err := strconv.ParseFloat(strValue, 64); err == nil {
			return f
		}
	}
	return value
}

func compareValues(a, b interface{}) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	switch aVal := a.(type) {
	case string:
		if bVal, ok := b.(string); ok {
			if aVal < bVal {
				return -1
			}
			if aVal > bVal {
				return 1
			}
			return 0
		}
	case int:
		if bVal, ok := b.(int); ok {
			return aVal - bVal
		}
	case float64:
		if bVal, ok := b.(float64); ok {
			if aVal < bVal {
				return -1
			}
			if aVal > bVal {
				return 1
			}
			return 0
		}
	}

	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	if aStr < bStr {
		return -1
	}
	if aStr > bStr {
		return 1
	}
	return 0
}
