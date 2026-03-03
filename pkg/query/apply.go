package query

import "fmt"

// ApplyQuery filters, sorts, and limits the given items based on QueryOptions.
func ApplyQuery(items []map[string]interface{}, queryOpts *QueryOptions) ([]map[string]interface{}, error) {
	result := items

	if queryOpts.Where != "" {
		var err error
		result, err = FilterItemsParallel(result, queryOpts, 0)
		if err != nil {
			return nil, fmt.Errorf("where clause error: %v", err)
		}
	}

	if queryOpts.HasOrderBy {
		var err error
		result, err = SortItems(result, queryOpts)
		if err != nil {
			return nil, err
		}
	}

	if queryOpts.HasLimit && queryOpts.Limit >= 0 && queryOpts.Limit < len(result) {
		result = result[:queryOpts.Limit]
	}

	return result, nil
}
