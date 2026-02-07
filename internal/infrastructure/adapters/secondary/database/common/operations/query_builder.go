package operations

import (
	"fmt"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
)

// SimpleQueryBuilder provides a simple implementation of QueryBuilder
type SimpleQueryBuilder struct {
	conditions []interfaces.QueryCondition
	orderBy    []interfaces.OrderByClause
	limit      int
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() interfaces.QueryBuilder {
	return &SimpleQueryBuilder{
		conditions: []interfaces.QueryCondition{},
		orderBy:    []interfaces.OrderByClause{},
		limit:      0,
	}
}

// Where adds a general condition to the query
func (q *SimpleQueryBuilder) Where(field string, operator string, value any) interfaces.QueryBuilder {
	q.conditions = append(q.conditions, interfaces.QueryCondition{
		Field:    field,
		Operator: operator,
		Value:    value,
	})
	return q
}

// WhereEqualTo adds an equality condition to the query
func (q *SimpleQueryBuilder) WhereEqualTo(field string, value any) interfaces.QueryBuilder {
	return q.Where(field, "==", value)
}

// WhereIn adds an "in" condition to the query
func (q *SimpleQueryBuilder) WhereIn(field string, values []any) interfaces.QueryBuilder {
	return q.Where(field, "in", values)
}

// OrderBy adds an order by clause to the query
func (q *SimpleQueryBuilder) OrderBy(field string, ascending bool) interfaces.QueryBuilder {
	q.orderBy = append(q.orderBy, interfaces.OrderByClause{
		Field:     field,
		Ascending: ascending,
	})
	return q
}

// Limit sets the maximum number of results to return
func (q *SimpleQueryBuilder) Limit(limit int) interfaces.QueryBuilder {
	q.limit = limit
	return q
}

// Build constructs the final query filter
func (q *SimpleQueryBuilder) Build() (interfaces.QueryFilter, error) {
	if len(q.conditions) == 0 {
		return interfaces.QueryFilter{}, fmt.Errorf("query must have at least one condition")
	}

	return interfaces.QueryFilter{
		Conditions: q.conditions,
		OrderBy:    q.orderBy,
		Limit:      q.limit,
	}, nil
}
