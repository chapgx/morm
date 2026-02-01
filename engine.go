package morm

import "fmt"

// TODO:(richard) may cange this later

// EngineData is data that each engine may need to execute work
type EngineData struct {
	table   string
	columns string
	dbname  string
	m       *MORM
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
