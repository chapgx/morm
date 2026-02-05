package morm

import (
	"fmt"
	"reflect"
)

// sqlite_createtable returns syntax to create at table in SQLITE
func sqlite_createtable_query(table, columns string) string {
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", table, columns)
}

//TODO: 2026-02-05 test this function

// sqlite_notag_column composes a field from a notag struct field
func sqlite_notag_column(field reflect.StructField, fieldname, tablename string) Field {
	kind := field.Type.Kind()
	switch kind {
	case reflect.Slice, reflect.Array, reflect.Map:
		k := field.Type.Elem().Kind()
		sqltype := sqlite_kind_to_sqltype(k)
		sqltype = fmt.Sprintf("value %s", sqltype)
		if kind == reflect.Bool {
			sqltype = fmt.Sprintf("%s check (value IN(0,1))", sqltype)
		}
		query := fmt.Sprintf("create table if not exists %s_%s (%s)", tablename, fieldname, sqltype)
		return Field{FieldType: ADJECENT_TABLE, query: query}
	default:
		sqltype := sqlite_kind_to_sqltype(kind)
		sqltype = fmt.Sprintf("%s %s", fieldname, sqltype)
		if kind == reflect.Bool {
			sqltype = fmt.Sprintf("%s check (%s IN(0,1))", sqltype, fieldname)
		}
		return NewField(sqltype)
	}
}

// sqllite_kind_to_sqltype takes a native k Kind and fieldname.
//
// Returns the sql type it should map to along the fieldname.
func sqlite_kind_to_sqltype(k reflect.Kind) string {
	switch k {
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Bool:
		return "integer"
	case reflect.String:
		return "text"
	//TODO:(richard) handle DATETIME
	default:
		panic(fmt.Sprintf("this type is not supported %s", k))
	}
}
