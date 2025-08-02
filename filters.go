package morm

import (
	"errors"
	"fmt"
	"strings"
)

type (
	FilterSeparator  string
	FilterComparison string
)

const (
	AND FilterSeparator = "and"
	OR  FilterSeparator = "or"
)

const (
	EQUAL           FilterComparison = "="
	NOT_EQUAL       FilterComparison = "<>"
	GREATER         FilterComparison = ">"
	GREATER_OR_EQ   FilterComparison = ">="
	LESS_THAN       FilterComparison = "<"
	LESS_THAN_OR_EQ FilterComparison = "<="
)

type Filter struct {
	items  []FilterItem
	groups []*FilterGroup
}

func (f *Filter) And(key string, c FilterComparison, val any) *Filter {
	i := FilterItem{key, val, c, AND}
	f.items = append(f.items, i)
	return f
}

func (f *Filter) Or(key string, c FilterComparison, val any) *Filter {
	i := FilterItem{key, val, c, OR}
	f.items = append(f.items, i)
	return f
}

func (f *Filter) Group() *FilterGroup {
	fg := FilterGroup{items: make([]FilterItem, 0)}
	if f.groups == nil {
		f.groups = make([]*FilterGroup, 0)
	}
	f.groups = append(f.groups, &fg)
	return &fg
}

func (f *Filter) WhereSQL() (string, error) {
	if len(f.items) == 1 && f.groups == nil {
		i := f.items[0]
		val, e := anytostr(i.val)
		if e != nil {
			return "", e
		}
		query := fmt.Sprintf("where %s%s%s", i.key, i.comparison, val)
		return query, nil
	}

	clause := []string{"where"}
	var e error
	for idx, i := range f.items {
		val, err := anytostr(i.val)
		if err != nil {
			e = err
			break
		}

		if idx == 0 {
			clause = append(clause, fmt.Sprintf("%s%s%s", i.key, i.comparison, val))
		} else {
			clause = append(clause, fmt.Sprintf("%s %s%s%s", i.separator, i.key, i.comparison, val))
		}
	}

	if e != nil {
		return "", e
	}

	query := strings.Join(clause, " ")
	if f.groups == nil {
		return query, nil
	}

	for _, g := range f.groups {
		sqlg, err := g.SQL()
		if err != nil {
			e = err
			break
		}
		query += "\n" + sqlg
	}

	if e != nil {
		return "", e
	}

	return query, nil
}

type FilterGroup struct {
	items []FilterItem
}

func (fg *FilterGroup) And(k string, c FilterComparison, v any) *FilterGroup {
	i := FilterItem{k, v, c, AND}
	fg.items = append(fg.items, i)
	return fg
}

func (fg *FilterGroup) Or(k string, c FilterComparison, v any) *FilterGroup {
	i := FilterItem{k, v, c, OR}
	fg.items = append(fg.items, i)
	return fg
}

func (fg *FilterGroup) SQL() (string, error) {
	if fg.items == nil {
		return "", errors.New("group items is <nil>")
	}

	if len(fg.items) == 0 {
		return "", errors.New("group items length is 0")
	}

	var clause []string
	var e error
	for idx, i := range fg.items {
		val, err := anytostr(i.val)
		if err != nil {
			e = err
			break
		}

		if idx == 0 {
			clause = append(clause, fmt.Sprintf("%s%s%s", i.key, i.comparison, val))
			continue
		}

		clause = append(clause, fmt.Sprintf("%s %s%s%s", i.separator, i.key, i.comparison, val))
	}

	if e != nil {
		return "", e
	}

	query := fmt.Sprintf("and (%s)", strings.Join(clause, " "))

	return query, nil
}

type FilterItem struct {
	key        string
	val        any
	comparison FilterComparison
	separator  FilterSeparator
}

func NewFilter() Filter {
	return Filter{items: make([]FilterItem, 0)}
}
