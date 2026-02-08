package listdata

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// SearchUtils provides utilities for full-text search across data items
type SearchUtils struct{}

// NewSearchUtils creates a new search utility instance
func NewSearchUtils() *SearchUtils {
	return &SearchUtils{}
}

// SearchResult represents an item with its search score and metadata
type SearchResult struct {
	Item       interface{}
	Score      float64
	Highlights map[string]string
}

// SearchItems performs full-text search on a slice of items
func (s *SearchUtils) SearchItems(
	items interface{},
	searchRequest *commonpb.SearchRequest,
) ([]*SearchResult, *commonpb.SearchMetrics) {
	if searchRequest == nil || searchRequest.Query == "" {
		return s.convertToSearchResults(items), &commonpb.SearchMetrics{
			TotalResults: int32(s.getSliceLength(items)),
			QueryTimeMs:  0,
		}
	}

	start := time.Now()

	sliceValue := reflect.ValueOf(items)
	if sliceValue.Kind() != reflect.Slice {
		return nil, &commonpb.SearchMetrics{}
	}

	var results []*SearchResult
	fieldMatchCounts := make(map[string]int32)

	// Process each item
	for i := 0; i < sliceValue.Len(); i++ {
		item := sliceValue.Index(i).Interface()
		result := s.searchSingleItem(item, searchRequest, fieldMatchCounts)
		if result != nil && result.Score > 0 {
			results = append(results, result)
		}
	}

	// Sort by score (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Apply max results limit
	if searchRequest.Options != nil && searchRequest.Options.MaxResults > 0 {
		maxResults := int(searchRequest.Options.MaxResults)
		if len(results) > maxResults {
			results = results[:maxResults]
		}
	}

	// Calculate metrics
	queryTime := time.Since(start).Milliseconds()
	topTerms := s.extractTopTerms(searchRequest.Query)

	metrics := &commonpb.SearchMetrics{
		TotalResults:     int32(len(results)),
		QueryTimeMs:      float64(queryTime),
		TopTerms:         topTerms,
		FieldMatchCounts: fieldMatchCounts,
	}

	return results, metrics
}

// searchSingleItem searches a single item and returns a SearchResult if it matches
func (s *SearchUtils) searchSingleItem(
	item interface{},
	searchRequest *commonpb.SearchRequest,
	fieldMatchCounts map[string]int32,
) *SearchResult {
	options := searchRequest.Options
	if options == nil {
		options = &commonpb.SearchOptions{}
	}

	queryTerms := s.tokenizeQuery(searchRequest.Query)
	if len(queryTerms) == 0 {
		return nil
	}

	var totalScore float64
	highlights := make(map[string]string)
	hasMatch := false

	// Get search fields (default to all string fields if not specified)
	searchFields := options.SearchFields
	if len(searchFields) == 0 {
		searchFields = s.getDefaultSearchFields(item)
	}

	// Search each field
	for _, fieldName := range searchFields {
		fieldValue := s.getFieldValue(item, fieldName)
		fieldText := s.fieldToText(fieldValue)

		if fieldText == "" {
			continue
		}

		fieldScore, highlight := s.searchInField(fieldText, queryTerms, options)
		if fieldScore > 0 {
			hasMatch = true
			fieldMatchCounts[fieldName]++

			// Apply field weight
			weight := 1.0
			if options.FieldWeights != nil {
				if w, exists := options.FieldWeights[fieldName]; exists {
					weight = w
				}
			}

			totalScore += fieldScore * weight

			// Add highlight if enabled
			if options.EnableHighlighting && highlight != "" {
				highlights[fieldName] = highlight
			}
		}
	}

	if !hasMatch {
		return nil
	}

	return &SearchResult{
		Item:       item,
		Score:      totalScore,
		Highlights: highlights,
	}
}

// searchInField searches for query terms within a single field
func (s *SearchUtils) searchInField(
	fieldText string,
	queryTerms []string,
	options *commonpb.SearchOptions,
) (float64, string) {
	fieldTextLower := strings.ToLower(fieldText)
	var score float64
	var highlights []string

	for _, term := range queryTerms {
		termLower := strings.ToLower(term)

		// Exact match
		if strings.Contains(fieldTextLower, termLower) {
			score += 1.0

			if options.EnableHighlighting {
				highlighted := s.highlightTerm(fieldText, term)
				if highlighted != "" {
					highlights = append(highlights, highlighted)
				}
			}
		} else if options.EnableFuzzy {
			// Fuzzy matching (simple implementation)
			fuzzyScore := s.fuzzyMatch(fieldTextLower, termLower)
			if fuzzyScore > 0.6 { // Threshold for fuzzy match
				score += fuzzyScore * 0.5 // Reduced score for fuzzy matches
			}
		}
	}

	highlight := ""
	if len(highlights) > 0 {
		highlight = highlights[0] // Return first highlight for simplicity
	}

	return score, highlight
}

// fuzzyMatch implements simple fuzzy matching using Levenshtein-like algorithm
func (s *SearchUtils) fuzzyMatch(text, term string) float64 {
	if len(term) == 0 {
		return 0
	}

	// Simple implementation: check if most characters of term are in text
	found := 0
	for _, char := range term {
		if strings.ContainsRune(text, char) {
			found++
		}
	}

	return float64(found) / float64(len(term))
}

// highlightTerm adds highlighting markup around matched terms
func (s *SearchUtils) highlightTerm(text, term string) string {
	// Simple highlighting with <mark> tags
	// In a real implementation, you'd want more sophisticated highlighting
	termLower := strings.ToLower(term)
	textLower := strings.ToLower(text)

	index := strings.Index(textLower, termLower)
	if index == -1 {
		return ""
	}

	// Extract context around the match
	start := index
	end := index + len(term)

	// Add context (up to 50 characters before and after)
	contextStart := start - 50
	if contextStart < 0 {
		contextStart = 0
	}
	contextEnd := end + 50
	if contextEnd > len(text) {
		contextEnd = len(text)
	}

	prefix := text[contextStart:start]
	match := text[start:end]
	suffix := text[end:contextEnd]

	return fmt.Sprintf("%s<mark>%s</mark>%s", prefix, match, suffix)
}

// tokenizeQuery breaks a search query into individual terms
func (s *SearchUtils) tokenizeQuery(query string) []string {
	// Simple tokenization - split by spaces and remove empty strings
	parts := strings.Fields(strings.TrimSpace(query))
	var terms []string

	for _, part := range parts {
		part = strings.Trim(part, ".,!?;:")
		if len(part) > 0 {
			terms = append(terms, part)
		}
	}

	return terms
}

// extractTopTerms extracts the most important terms from a query
func (s *SearchUtils) extractTopTerms(query string) []string {
	terms := s.tokenizeQuery(query)

	// Simple implementation - return unique terms, filter out common words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
	}

	var topTerms []string
	seen := make(map[string]bool)

	for _, term := range terms {
		termLower := strings.ToLower(term)
		if !stopWords[termLower] && !seen[termLower] && len(term) > 2 {
			topTerms = append(topTerms, term)
			seen[termLower] = true
		}
	}

	return topTerms
}

// getDefaultSearchFields returns default searchable fields for an item
func (s *SearchUtils) getDefaultSearchFields(item interface{}) []string {
	var fields []string

	val := reflect.ValueOf(item)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fields
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Include string fields and common text fields
		if s.isSearchableField(field, fieldVal) {
			fields = append(fields, s.toSnakeCase(field.Name))
		}
	}

	return fields
}

// isSearchableField determines if a field should be included in default search
func (s *SearchUtils) isSearchableField(field reflect.StructField, value reflect.Value) bool {
	// Skip unexported fields
	if !field.IsExported() {
		return false
	}

	fieldName := strings.ToLower(field.Name)

	// Include common text fields
	textFields := []string{"name", "title", "description", "content", "text", "email"}
	for _, textField := range textFields {
		if strings.Contains(fieldName, textField) {
			return true
		}
	}

	// Include string types
	kind := value.Kind()
	if kind == reflect.String {
		return true
	}
	if kind == reflect.Ptr && value.Type().Elem().Kind() == reflect.String {
		return true
	}

	return false
}

// getFieldValue extracts field value using dot notation (similar to other utils)
func (s *SearchUtils) getFieldValue(item interface{}, fieldPath string) interface{} {
	if item == nil {
		return nil
	}

	val := reflect.ValueOf(item)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	parts := strings.Split(fieldPath, ".")

	for _, part := range parts {
		if val.Kind() == reflect.Struct {
			fieldName := s.toCamelCase(part)
			field := val.FieldByName(fieldName)
			if !field.IsValid() {
				return nil
			}

			if field.Kind() == reflect.Ptr {
				if field.IsNil() {
					return nil
				}
				val = field.Elem()
			} else {
				val = field
			}
		} else {
			return nil
		}
	}

	return val.Interface()
}

// fieldToText converts a field value to searchable text
func (s *SearchUtils) fieldToText(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case *string:
		if v != nil {
			return *v
		}
	default:
		return fmt.Sprintf("%v", value)
	}

	return ""
}

// Helper functions

func (s *SearchUtils) toCamelCase(str string) string {
	parts := strings.Split(str, "_")
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	if len(result) > 0 {
		result = strings.ToUpper(result[:1]) + result[1:]
	}
	return result
}

func (s *SearchUtils) toSnakeCase(str string) string {
	var result []rune
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r+'a'-'A')
	}
	return string(result)
}

func (s *SearchUtils) getSliceLength(items interface{}) int {
	sliceValue := reflect.ValueOf(items)
	if sliceValue.Kind() != reflect.Slice {
		return 0
	}
	return sliceValue.Len()
}

func (s *SearchUtils) convertToSearchResults(items interface{}) []*SearchResult {
	sliceValue := reflect.ValueOf(items)
	if sliceValue.Kind() != reflect.Slice {
		return nil
	}

	var results []*SearchResult
	for i := 0; i < sliceValue.Len(); i++ {
		item := sliceValue.Index(i).Interface()
		results = append(results, &SearchResult{
			Item:       item,
			Score:      1.0, // Default score when no search is performed
			Highlights: make(map[string]string),
		})
	}

	return results
}
