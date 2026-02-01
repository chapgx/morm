package morm

import "fmt"

// sqlite_createtable returns syntax to create at table in SQLITE
func sqlite_createtable_query(table, columns string) string {
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", table, columns)
}
