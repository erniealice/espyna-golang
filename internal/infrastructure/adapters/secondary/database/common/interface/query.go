package interfaces

import "fmt"

// QueryBuilder provides a builder pattern for constructing database queries
type QueryBuilder interface {
	Where(field string, operator string, value any) QueryBuilder
	WhereEqualTo(field string, value any) QueryBuilder
	WhereIn(field string, values []any) QueryBuilder
	OrderBy(field string, ascending bool) QueryBuilder
	Limit(limit int) QueryBuilder
	Build() (QueryFilter, error)
}

// QueryFilter represents a structured query filter
type QueryFilter struct {
	Conditions []QueryCondition
	OrderBy    []OrderByClause
	Limit      int
}

// QueryCondition represents a single query condition
type QueryCondition struct {
	Field    string
	Operator string
	Value    any
}

// OrderByClause represents a single order by clause
type OrderByClause struct {
	Field     string
	Ascending bool
}

// CompositeKeyQuery provides a helper for composite key queries
type CompositeKeyQuery struct {
	Keys map[string]any
}

// NewCompositeKeyQuery creates a new composite key query
func NewCompositeKeyQuery() *CompositeKeyQuery {
	return &CompositeKeyQuery{
		Keys: make(map[string]any),
	}
}

// AddKey adds a key-value pair to the composite key query
func (c *CompositeKeyQuery) AddKey(key string, value any) *CompositeKeyQuery {
	c.Keys[key] = value
	return c
}

// ToQueryBuilder converts the composite key query to a QueryBuilder
func (c *CompositeKeyQuery) ToQueryBuilder() QueryBuilder {
	builder := NewQueryBuilder()
	for key, value := range c.Keys {
		builder.WhereEqualTo(key, value)
	}
	return builder
}

// SimpleQueryBuilder provides a simple implementation of QueryBuilder
type SimpleQueryBuilder struct {
	conditions []QueryCondition
	orderBy    []OrderByClause
	limit      int
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() QueryBuilder {
	return &SimpleQueryBuilder{
		conditions: []QueryCondition{},
		orderBy:    []OrderByClause{},
		limit:      0,
	}
}

// Where adds a general condition to the query
func (q *SimpleQueryBuilder) Where(field string, operator string, value any) QueryBuilder {
	q.conditions = append(q.conditions, QueryCondition{
		Field:    field,
		Operator: operator,
		Value:    value,
	})
	return q
}

// WhereEqualTo adds an equality condition to the query
func (q *SimpleQueryBuilder) WhereEqualTo(field string, value any) QueryBuilder {
	return q.Where(field, "==", value)
}

// WhereIn adds an "in" condition to the query
func (q *SimpleQueryBuilder) WhereIn(field string, values []any) QueryBuilder {
	return q.Where(field, "in", values)
}

// OrderBy adds an order by clause to the query
func (q *SimpleQueryBuilder) OrderBy(field string, ascending bool) QueryBuilder {
	q.orderBy = append(q.orderBy, OrderByClause{
		Field:     field,
		Ascending: ascending,
	})
	return q
}

// Limit sets the maximum number of results to return
func (q *SimpleQueryBuilder) Limit(limit int) QueryBuilder {
	q.limit = limit
	return q
}

// Build constructs the final query filter
func (q *SimpleQueryBuilder) Build() (QueryFilter, error) {
	if len(q.conditions) == 0 {
		return QueryFilter{}, fmt.Errorf("query must have at least one condition")
	}

	return QueryFilter{
		Conditions: q.conditions,
		OrderBy:    q.orderBy,
		Limit:      q.limit,
	}, nil
}
