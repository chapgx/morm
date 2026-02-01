package morm

import "fmt"

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
