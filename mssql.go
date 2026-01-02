package morm

import (
	"fmt"
)

type MSSQLInfo struct {
	version       string
	edition       string
	engineEdition int
}

func (i MSSQLInfo) Version() string { return i.version }

func (i MSSQLInfo) Edition() string { return i.edition }

func (i MSSQLInfo) EngineEdition() int { return i.engineEdition }

func (i MSSQLInfo) IsNil() bool {
	return i.version == ""
}

const mssql_info_query = `
SELECT
    SERVERPROPERTY('ProductVersion')     AS product_version,
    -- SERVERPROPERTY('ProductLevel')       AS product_level,   -- RTM / CU
    SERVERPROPERTY('Edition')            AS edition,
    SERVERPROPERTY('EngineEdition')      AS engine_edition,
    -- SERVERPROPERTY('ProductUpdateLevel') AS cu_level,
    -- SERVERPROPERTY('ProductUpdateReference') AS cu_kb;
	`

func mssql_server_info(m *MORM) (MSSQLInfo, error) {
	var info MSSQLInfo

	if m.engine != SQLServer {
		return info, fmt.Errorf("expected mssql server engine but got %s", m.engine)
	}

	if !m.connected {
		e := m.connect()
		if e != nil {
			return info, e
		}
		defer m.Close()
	}

	row := m.db.QueryRow(mssql_info_query)

	e := row.Scan(&info.version, &info.edition, &info.engineEdition)
	if e != nil {
		return info, e
	}

	return info, nil
}

// mssql_insert prepares pre statements if needed before the insert query
func mssql_insert(m *MORM) (string, error) {
	if m.engine != SQLServer {
		return "", fmt.Errorf("expected SQLServer engine but got %s", m.engine)
	}

	if m.databasename == "" {
		return "", nil
	}

	return fmt.Sprintf("USE %s;\n\n", m.databasename), nil
}
