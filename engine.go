package morm

import "fmt"

func sqlite_createtable(table, columns string) string {
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", table, columns)
}

func mssql_createtable(table, columns, dbname string, m *MORM) (string, error) {

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
