package query

import (
	"fmt"
	"strings"
)

type QueryParams struct {
	SelectColumns []string
	Where         []WhereCondition
	OrderBy       []OrderByClause
	Limit         int
	Offset        int
	Args          []interface{}
}

type WhereCondition struct {
	Column   string
	Operator string
	Value    interface{}
}

type OrderByClause struct {
	Column string
	Desc   bool
}

func (p *QueryParams) GetSelectClause() string {
	if len(p.SelectColumns) == 0 {
		return "*"
	}
	return strings.Join(p.SelectColumns, ", ")
}

func (p *QueryParams) GetWhereClause() string {
	if len(p.Where) == 0 {
		return ""
	}
	conditions := make([]string, len(p.Where))
	for i, w := range p.Where {
		conditions[i] = fmt.Sprintf("%s %s ?", w.Column, w.Operator)
		p.Args = append(p.Args, w.Value)
	}
	return strings.Join(conditions, " AND ")
}

func (p *QueryParams) GetOrderByClause() string {
	if len(p.OrderBy) == 0 {
		return ""
	}
	parts := make([]string, len(p.OrderBy))
	for i, o := range p.OrderBy {
		if o.Desc {
			parts[i] = fmt.Sprintf("%s DESC", o.Column)
		} else {
			parts[i] = o.Column
		}
	}
	return strings.Join(parts, ", ")
}

func (p *QueryParams) GetLimitClause() string {
	if p.Limit <= 0 {
		return ""
	}
	if p.Offset > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", p.Limit, p.Offset)
	}
	return fmt.Sprintf("LIMIT %d", p.Limit)
}

func (p *QueryParams) GetArgs() []interface{} {
	return p.Args
}

func (p *QueryParams) AddWhere(column, operator string, value interface{}) {
	p.Where = append(p.Where, WhereCondition{
		Column:   column,
		Operator: operator,
		Value:    value,
	})
}

func (p *QueryParams) AddOrderBy(column string, desc bool) {
	p.OrderBy = append(p.OrderBy, OrderByClause{
		Column: column,
		Desc:   desc,
	})
}
