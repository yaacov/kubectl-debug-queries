package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var selectRegexp = regexp.MustCompile(`(?i)^(?:(sum|len|any|all)\s*\(?\s*([^)\s]+)\s*\)?|(.+?))\s*(?:as\s+(.+))?$`)

func parseSelectClause(selectClause string) []SelectOption {
	var opts []SelectOption
	for _, raw := range strings.Split(selectClause, ",") {
		field := strings.TrimSpace(raw)
		if field == "" {
			continue
		}
		if m := selectRegexp.FindStringSubmatch(field); m != nil {
			reducer := strings.ToLower(m[1])
			expr := m[2]
			if expr == "" {
				expr = m[3]
			}
			alias := m[4]
			if alias == "" {
				alias = expr
			}
			if !strings.HasPrefix(expr, ".") && !strings.HasPrefix(expr, "{") {
				expr = "." + expr
			}
			opts = append(opts, SelectOption{
				Field:   expr,
				Alias:   alias,
				Reducer: reducer,
			})
		}
	}
	return opts
}

func parseOrderByClause(orderByClause string, selectOpts []SelectOption) []OrderOption {
	var orderOpts []OrderOption

	for _, rawField := range strings.Split(orderByClause, ",") {
		fieldStr := strings.TrimSpace(rawField)
		if fieldStr == "" {
			continue
		}

		parts := strings.Fields(fieldStr)
		descending := false
		last := parts[len(parts)-1]
		if strings.EqualFold(last, "desc") {
			descending = true
			parts = parts[:len(parts)-1]
		} else if strings.EqualFold(last, "asc") {
			parts = parts[:len(parts)-1]
		}

		name := strings.Join(parts, " ")
		if !strings.HasPrefix(name, ".") && !strings.HasPrefix(name, "{") {
			name = "." + name
		}

		var selOpt SelectOption
		found := false
		for _, sel := range selectOpts {
			if sel.Field == name || sel.Alias == strings.TrimPrefix(name, ".") {
				selOpt = sel
				found = true
				break
			}
		}
		if !found {
			selOpt = SelectOption{
				Field: name,
				Alias: strings.TrimPrefix(name, "."),
			}
		}

		orderOpts = append(orderOpts, OrderOption{
			Field:      selOpt,
			Descending: descending,
		})
	}

	return orderOpts
}

var validQueryPrefixes = []string{"select ", "where ", "order by ", "sort by ", "limit "}

func hasQueryKeywordPrefix(query string) bool {
	lower := strings.ToLower(strings.TrimSpace(query))
	for _, prefix := range validQueryPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// ParseQueryString parses a TSL query string into its component parts.
// Supports: SELECT fields WHERE condition ORDER BY field LIMIT n
func ParseQueryString(query string) (*QueryOptions, error) {
	options := &QueryOptions{
		Limit: -1,
	}

	if query == "" {
		return options, nil
	}

	// Bare filter expressions (no keyword prefix) get "where " prepended.
	if !hasQueryKeywordPrefix(query) {
		query = "where " + query
	}

	if err := ValidateQuerySyntax(query); err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)

	selectIndex := strings.Index(queryLower, "select ")
	whereIndex := strings.Index(queryLower, "where ")
	limitIndex := strings.Index(queryLower, "limit ")

	orderByIndex := strings.Index(queryLower, "order by ")
	if orderByIndex == -1 {
		orderByIndex = strings.Index(queryLower, "sort by ")
	}

	if selectIndex >= 0 {
		selectEnd := len(query)
		if whereIndex > selectIndex {
			selectEnd = whereIndex
		} else if orderByIndex > selectIndex {
			selectEnd = orderByIndex
		} else if limitIndex > selectIndex {
			selectEnd = limitIndex
		}

		selectClause := strings.TrimSpace(query[selectIndex+7 : selectEnd])
		options.Select = parseSelectClause(selectClause)
		options.HasSelect = len(options.Select) > 0
	}

	if whereIndex >= 0 {
		whereEnd := len(query)
		if orderByIndex > whereIndex {
			whereEnd = orderByIndex
		} else if limitIndex > whereIndex {
			whereEnd = limitIndex
		}

		options.Where = strings.TrimSpace(query[whereIndex+6 : whereEnd])
	}

	if orderByIndex >= 0 {
		orderByEnd := len(query)
		if limitIndex > orderByIndex {
			orderByEnd = limitIndex
		}

		orderByClause := strings.TrimSpace(query[orderByIndex+8 : orderByEnd])
		options.OrderBy = parseOrderByClause(orderByClause, options.Select)
		options.HasOrderBy = len(options.OrderBy) > 0
	}

	limitRegex := regexp.MustCompile(`(?i)limit\s+(\d+)`)
	limitMatches := limitRegex.FindStringSubmatch(query)
	if len(limitMatches) > 1 {
		limit, err := strconv.Atoi(limitMatches[1])
		if err != nil {
			return nil, fmt.Errorf("invalid limit value: %v", err)
		}
		options.Limit = limit
		options.HasLimit = true
	}

	return options, nil
}
