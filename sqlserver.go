package morm

import (
	"fmt"
	"reflect"
)

func mssql_createtable_query(table, columns, dbname string, m *MORM) (string, error) {

	var query string
	if dbname != "" {
		create_db_query := `
		IF DB_ID(N'%s') IS NULL
		BEGIN
			CREATE DATABASE %s;
		END;
		`
		create_db_query = fmt.Sprintf(create_db_query, dbname, dbname)
		_, e := m.Exec(create_db_query)
		if e != nil {
			return "", e
		}

		query = fmt.Sprintf("USE %s;\n\n", dbname)
	}

	query += `
IF OBJECT_ID('%s', 'U') IS NULL
BEGIN
    CREATE TABLE %s (%s)
END
	`

	query = fmt.Sprintf(query, table, table, columns)
	return query, nil
}

// mssql_notag_column composes a no tag (a field without the morm tag)
func mssql_notag_column(field reflect.StructField, fieldname, tablename string) Field {
	switch field.Type.Kind() {
	case reflect.Int8:
		return NewField(fmt.Sprintf("%s TINYINT", fieldname))
	case reflect.Int16:
		return NewField(fmt.Sprintf("%s SMALLINT", fieldname))
	case reflect.Int32:
		return NewField(fmt.Sprintf("%s INT", fieldname))
	case reflect.Int64:
		return NewField(fmt.Sprintf("%s BIGINT", fieldname))
	case reflect.String:
		return NewField(fmt.Sprintf("%s varchar(max)", fieldname))
	case reflect.Bool:
		return NewField(fmt.Sprintf("%s BIT", fieldname))
	case reflect.Array, reflect.Slice, reflect.Map:
		//TODO: handle arrays and slices by creatin an adjecent table by creating am adjecent table
		return Field{}
	default:
		panic(fmt.Sprintf("this type is not supported %s", field.Type.Kind()))
	}

}
