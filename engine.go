package morm

import (
	"fmt"
	"reflect"
)

// TODO:(richard) may change this later

// EngineData is data that each engine may need to do work
type EngineData struct {
	table   string
	columns string
	dbname  string
	m       *MORM
}

type NoTagType int

const (
	COLUMN NoTagType = iota + 1
	ADJECENT_TABLE
)

// Field is the in memor representation of what a column could resolve to
//
// For now it could be just a column or an adjecent table which is a table with
// a relation to another  table
type Field struct {
	// The type of the field a column or an adjecent table
	FieldType NoTagType

	// the query to execute
	query string

	// in case of a adjecent table type the name is attached as well
	//NOTE: may not be needed
	parent_table_field string
}

func NewField(query string) Field {
	return Field{FieldType: COLUMN, query: query}
}

// create_table_query composes the create table query
//
// may panic on SQLServer since it executes a create database if it does not exists
func create_table_query(data EngineData) (string, error) {
	switch data.m.engine {
	case SQLITE:
		query := sqlite_createtable_query(data.table, data.columns)
		return query, nil
	case SQLServer:
		query, e := mssql_createtable_query(data.table, data.columns, data.dbname, data.m)
		return query, e
	default:
		panic(fmt.Sprintf("engine %s is not supported", data.m.engine))
	}
}

// notag_column creates the sql syntax with the correct type based on the struct field type
//
// is possible it returns a column or a table query depending on the stuct field data type
func notag_column(field reflect.StructField, fieldname string, m *MORM, tablename string) Field {
	if m == nil {
		panic("morm client is nil")
	}

	switch m.engine {
	case SQLITE:
		return sqlite_notag_column(field, fieldname, tablename)
	case SQLServer:
		return mssql_notag_column(field, fieldname, tablename)
	default:
		panic(fmt.Sprintf("driver %s is not supported", m.engine))
	}

}
