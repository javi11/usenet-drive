package utils

import "fmt"

type sqlFilterBuilder struct {
	whereClause   string
	orderByClause string
}

func NewSqlFilterBuilder() *sqlFilterBuilder {
	return &sqlFilterBuilder{}
}

func (s *sqlFilterBuilder) AddFilter(name string, filter Filter) string {
	s.whereClause += fmt.Sprintf("%s LIKE ? AND ", name)

	return s.buildSqlFilter(name, filter)
}

func (s *sqlFilterBuilder) AddSortBy(name string, direction SortByDirection) {
	s.orderByClause += s.buildSqlSortBy(name, direction)
}

func (s *sqlFilterBuilder) Build() string {
	sql := ""
	if s.whereClause != "" {
		sql += fmt.Sprintf("WHERE %s", s.whereClause[:len(s.whereClause)-5])
	}

	if s.orderByClause != "" {
		sql += fmt.Sprintf(" ORDER BY %s", s.orderByClause[:len(s.orderByClause)-2])
	}

	return sql
}

func (s *sqlFilterBuilder) buildSqlFilter(name string, filter Filter) string {
	arg := ""
	switch filter.Mode {
	case "startsWith":
		arg = fmt.Sprintf("%%%s", filter.Value)
	case "endsWith":
		arg = fmt.Sprintf("%s%%", filter.Value)
	default:
		arg = fmt.Sprintf("%%%s%%", filter.Value)
	}

	return arg
}

func (s *sqlFilterBuilder) buildSqlSortBy(name string, direction SortByDirection) string {
	orderByClause := ""
	switch direction {
	case "desc":
		orderByClause += fmt.Sprintf("%s DESC, ", name)
	default:
		orderByClause += fmt.Sprintf("%s ASC, ", name)
	}

	return orderByClause
}
