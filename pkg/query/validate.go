package query

import (
	"fmt"
	"regexp"
	"strings"
)

// queryKeyword represents a SQL-like keyword in queries.
type queryKeyword struct {
	Name     string
	Pattern  *regexp.Regexp
	Position int // expected order: 1=SELECT, 2=WHERE, 3=ORDER BY, 4=LIMIT
}

var keywords = []queryKeyword{
	{"SELECT", regexp.MustCompile(`(?i)\bselect\b`), 1},
	{"WHERE", regexp.MustCompile(`(?i)\bwhere\b`), 2},
	{"ORDER BY", regexp.MustCompile(`(?i)\border\s+by\b`), 3},
	{"SORT BY", regexp.MustCompile(`(?i)\bsort\s+by\b`), 3},
	{"LIMIT", regexp.MustCompile(`(?i)\blimit\b`), 4},
}

var typoCorrections = map[string]string{
	"selct": "SELECT", "slect": "SELECT",
	"wher": "WHERE", "were": "WHERE", "whre": "WHERE",
	"limt": "LIMIT", "lmit": "LIMIT",
	"oder": "ORDER", "ordr": "ORDER",
	"srot": "SORT", "sotr": "SORT",
}

// ValidationError represents a query syntax error with suggestions.
type ValidationError struct {
	Type       string
	Message    string
	Suggestion string
	Position   int
	FoundText  string
}

func (e ValidationError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("%s: %s. Suggestion: %s", e.Type, e.Message, e.Suggestion)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

type keywordOccurrence struct {
	Keyword   queryKeyword
	Count     int
	Positions []int
}

// ValidateQuerySyntax checks the query for typos, duplicate/conflicting keywords,
// clause ordering, and empty clauses.
func ValidateQuerySyntax(query string) error {
	if query == "" {
		return nil
	}

	if err := checkTypos(query); err != nil {
		return err
	}

	occurrences := countKeywordOccurrences(query)

	if err := checkDuplicateKeywords(occurrences); err != nil {
		return err
	}
	if err := checkClauseOrdering(occurrences); err != nil {
		return err
	}
	if err := checkConflictingKeywords(occurrences); err != nil {
		return err
	}
	if err := checkEmptyClauses(query, occurrences); err != nil {
		return err
	}

	return nil
}

func checkTypos(query string) error {
	queryLower := strings.ToLower(query)
	words := regexp.MustCompile(`\b\w+\b`).FindAllString(queryLower, -1)

	for _, word := range words {
		if correction, found := typoCorrections[word]; found && word != strings.ToLower(correction) {
			isValidKeyword := false
			for _, kw := range keywords {
				if kw.Pattern.MatchString(word) {
					isValidKeyword = true
					break
				}
			}
			if !isValidKeyword {
				return ValidationError{
					Type:       "Keyword Typo",
					Message:    fmt.Sprintf("Unrecognized keyword '%s'", word),
					Suggestion: fmt.Sprintf("Did you mean '%s'?", correction),
					FoundText:  word,
				}
			}
		}
	}
	return nil
}

func countKeywordOccurrences(query string) []keywordOccurrence {
	var occurrences []keywordOccurrence
	for _, keyword := range keywords {
		matches := keyword.Pattern.FindAllStringIndex(query, -1)
		if len(matches) > 0 {
			positions := make([]int, len(matches))
			for i, match := range matches {
				positions[i] = match[0]
			}
			occurrences = append(occurrences, keywordOccurrence{
				Keyword:   keyword,
				Count:     len(matches),
				Positions: positions,
			})
		}
	}
	return occurrences
}

func checkDuplicateKeywords(occurrences []keywordOccurrence) error {
	for _, occ := range occurrences {
		if occ.Count > 1 {
			return ValidationError{
				Type:       "Duplicate Keyword",
				Message:    fmt.Sprintf("Keyword '%s' appears %d times", occ.Keyword.Name, occ.Count),
				Suggestion: fmt.Sprintf("Use '%s' only once in your query", occ.Keyword.Name),
			}
		}
	}
	return nil
}

func checkClauseOrdering(occurrences []keywordOccurrence) error {
	positionMap := make(map[int]queryKeyword)
	for _, occ := range occurrences {
		if len(occ.Positions) > 0 {
			positionMap[occ.Positions[0]] = occ.Keyword
		}
	}

	var positions []int
	for pos := range positionMap {
		positions = append(positions, pos)
	}

	for i := 0; i < len(positions); i++ {
		for j := 0; j < len(positions)-1-i; j++ {
			if positions[j] > positions[j+1] {
				positions[j], positions[j+1] = positions[j+1], positions[j]
			}
		}
	}

	lastPosition := 0
	for _, pos := range positions {
		keyword := positionMap[pos]
		if keyword.Position < lastPosition {
			return ValidationError{
				Type:       "Invalid Clause Order",
				Message:    fmt.Sprintf("'%s' appears after a clause that should come later", keyword.Name),
				Suggestion: "Use the order: SELECT → WHERE → ORDER BY/SORT BY → LIMIT",
			}
		}
		lastPosition = keyword.Position
	}
	return nil
}

func checkConflictingKeywords(occurrences []keywordOccurrence) error {
	hasOrderBy := false
	hasSortBy := false
	for _, occ := range occurrences {
		switch occ.Keyword.Name {
		case "ORDER BY":
			hasOrderBy = true
		case "SORT BY":
			hasSortBy = true
		}
	}
	if hasOrderBy && hasSortBy {
		return ValidationError{
			Type:       "Conflicting Keywords",
			Message:    "Cannot use both 'ORDER BY' and 'SORT BY' in the same query",
			Suggestion: "Choose either 'ORDER BY' or 'SORT BY', not both",
		}
	}
	return nil
}

func checkEmptyClauses(query string, occurrences []keywordOccurrence) error {
	queryLower := strings.ToLower(query)

	for _, occ := range occurrences {
		if len(occ.Positions) == 0 {
			continue
		}

		pos := occ.Positions[0]
		keywordName := strings.ToLower(occ.Keyword.Name)

		keywordEndPos := pos + len(keywordName)
		if strings.Contains(keywordName, " ") {
			pattern := strings.ReplaceAll(keywordName, " ", `\s+`)
			regex := regexp.MustCompile(`(?i)` + pattern)
			match := regex.FindStringIndex(queryLower[pos:])
			if match != nil {
				keywordEndPos = pos + match[1]
			}
		}

		afterKeyword := query[keywordEndPos:]
		nextKeywordPos := len(afterKeyword)

		for _, nextOcc := range occurrences {
			for _, nextPos := range nextOcc.Positions {
				if nextPos > keywordEndPos {
					relativePos := nextPos - keywordEndPos
					if relativePos < nextKeywordPos {
						nextKeywordPos = relativePos
					}
				}
			}
		}

		clauseContent := strings.TrimSpace(afterKeyword[:nextKeywordPos])
		if clauseContent == "" {
			return ValidationError{
				Type:       "Empty Clause",
				Message:    fmt.Sprintf("'%s' keyword found but no content follows", occ.Keyword.Name),
				Suggestion: fmt.Sprintf("Add content after '%s' or remove the keyword", occ.Keyword.Name),
			}
		}
	}
	return nil
}
