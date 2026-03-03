package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// GetValueByPathString gets a value from an object using a string path.
// Supports dot notation (e.g. "metadata.name") and array indexing (e.g. "items[0].name").
// For flat maps, also supports flexible matching: case-insensitive and underscore-to-space.
func GetValueByPathString(obj interface{}, path string) (interface{}, error) {
	path = strings.TrimPrefix(path, "{{")
	path = strings.TrimSuffix(path, "}}")
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, ".")

	var parts []string
	var currentPart strings.Builder
	insideBrackets := false

	for _, char := range path {
		switch char {
		case '[':
			insideBrackets = true
			currentPart.WriteRune(char)
		case ']':
			insideBrackets = false
			currentPart.WriteRune(char)
		case '.':
			if insideBrackets {
				currentPart.WriteRune(char)
			} else {
				parts = append(parts, currentPart.String())
				currentPart.Reset()
			}
		default:
			currentPart.WriteRune(char)
		}
	}
	if currentPart.Len() > 0 {
		parts = append(parts, currentPart.String())
	}

	return getValueByPath(obj, parts)
}

func getValueByPath(obj interface{}, pathParts []string) (interface{}, error) {
	if len(pathParts) == 0 {
		return obj, nil
	}
	if obj == nil {
		return nil, fmt.Errorf("cannot access %s on nil value", strings.Join(pathParts, "."))
	}

	part := pathParts[0]
	remainingParts := pathParts[1:]

	arrayIndex := -1
	mapKey := ""
	isWildcard := false

	wildcardMatch := regexp.MustCompile(`(.*)\[\*\]$`).FindStringSubmatch(part)
	arrayMatch := regexp.MustCompile(`(.*)\[(\d+)\]$`).FindStringSubmatch(part)
	mapMatch := regexp.MustCompile(`(.*)\[([^\]]+)\]$`).FindStringSubmatch(part)

	if len(wildcardMatch) == 2 {
		part = wildcardMatch[1]
		isWildcard = true
	} else if len(arrayMatch) == 3 {
		part = arrayMatch[1]
		index, err := strconv.Atoi(arrayMatch[2])
		if err != nil {
			return nil, fmt.Errorf("invalid array index in path: %s", part)
		}
		arrayIndex = index
	} else if len(mapMatch) == 3 {
		part = mapMatch[1]
		mapKey = strings.Trim(mapMatch[2], `"'`)
	}

	switch objTyped := obj.(type) {
	case map[string]interface{}:
		value, exists := flexibleMapLookup(objTyped, part)
		if !exists {
			return nil, nil
		}

		if isWildcard {
			arr, ok := value.([]interface{})
			if !ok {
				return nil, fmt.Errorf("cannot apply wildcard to non-array value: %s", part)
			}
			var results []interface{}
			for _, item := range arr {
				result, err := getValueByPath(item, remainingParts)
				if err == nil && result != nil {
					if resultArray, isArray := result.([]interface{}); isArray {
						results = append(results, resultArray...)
					} else {
						results = append(results, result)
					}
				}
			}
			return results, nil
		}

		if arrayIndex >= 0 {
			if arr, ok := value.([]interface{}); ok {
				if arrayIndex >= len(arr) {
					return nil, fmt.Errorf("array index out of bounds: %d", arrayIndex)
				}
				value = arr[arrayIndex]
			} else {
				return nil, fmt.Errorf("cannot apply array index to non-array value: %s", part)
			}
		}

		if mapKey != "" {
			if m, ok := value.(map[string]interface{}); ok {
				mapValue, exists := m[mapKey]
				if !exists {
					return nil, nil
				}
				value = mapValue
			} else {
				return nil, fmt.Errorf("cannot apply map key to non-map value: %s", part)
			}
		}

		if len(remainingParts) == 0 {
			return value, nil
		}

		if arr, ok := value.([]interface{}); ok {
			var results []interface{}
			for _, item := range arr {
				result, err := getValueByPath(item, remainingParts)
				if err == nil && result != nil {
					if resultArray, isArray := result.([]interface{}); isArray {
						results = append(results, resultArray...)
					} else {
						results = append(results, result)
					}
				}
			}
			return results, nil
		}

		return getValueByPath(value, remainingParts)

	default:
		return nil, fmt.Errorf("cannot access property %s on non-object value", part)
	}
}

// flexibleMapLookup tries exact match, then case-insensitive,
// then underscore→space (case-insensitive). This allows users to write
// "Last_Seen" to match column names like "Last Seen".
func flexibleMapLookup(m map[string]interface{}, key string) (interface{}, bool) {
	if v, ok := m[key]; ok {
		return v, true
	}

	keyLower := strings.ToLower(key)
	for k, v := range m {
		if strings.ToLower(k) == keyLower {
			return v, true
		}
	}

	keyWithSpaces := strings.ReplaceAll(key, "_", " ")
	for k, v := range m {
		if strings.EqualFold(k, keyWithSpaces) {
			return v, true
		}
	}

	return nil, false
}

// GetValue retrieves a value by name (or alias) from obj using JSONPath,
// then applies a reducer if one is specified in selectOpts.
func GetValue(obj interface{}, name string, selectOpts []SelectOption) (interface{}, error) {
	path := name
	var reducer string

	for _, opt := range selectOpts {
		if opt.Alias == name {
			path = opt.Field
			reducer = opt.Reducer
			break
		}
	}

	val, err := GetValueByPathString(obj, path)
	if err != nil {
		return nil, err
	}

	if reducer != "" {
		reduced, err := applyReducer(val, reducer)
		if err != nil {
			return nil, fmt.Errorf("failed to apply reducer %q: %v", reducer, err)
		}
		return reduced, nil
	}

	return val, nil
}
