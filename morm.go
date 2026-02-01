package morm

import (
	"database/sql"
	"errors"
	"fmt"
	. "github.com/chapgx/assert/v2"
	"reflect"
	"strings"
)

// NOTE: for dev only
var _queryHistory []string

func PrintQueryHistory() {
	for _, query := range _queryHistory {
		fmt.Println(query)
	}
}

type DBInfo interface {
	Version() string
	Edition() string
	EngineEdition() int
}

type FnConnect func() error

type MORM struct {
	db           *sql.DB
	connected    bool
	engine       ENGINE
	connect      FnConnect
	databasename string
	info         DBInfo
}

// GetDatabaseName returns the databasename is any
func (m *MORM) GetDatabaseName() string { return m.databasename }

// Close() closes the database connection and resets [MORM]
func (m *MORM) Close() error {
	if m == nil {
		return ErrDefaultClientIsNil
	}

	if m.db == nil {
		return ErrDBIsNil
	}

	if !m.connected {
		return errors.New("database is not connected")
	}

	return m.db.Close()
}

// CreateTable creates a table base on the model and optional tablename
func (m *MORM) CreateTable(model any, tablename string) error {
	createdepth++

	t := pulltype(model)

	if tablename == "" {
		tablename = strings.ToLower(t.Name())
		if !strings.HasSuffix(tablename, "s") {
			tablename += "s"
		}
	}

	var columns []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		mormtag := gettag(field)

		// note: untagged are added as text with their field name
		if mormtag.IsEmpty() {
			//BUG: adjecent structs are not being created with a  link or foreign key

			if field.Type.Kind() == reflect.Struct {
				e := m.CreateTable(field.Type, "")
				if e != nil {
					panic(e)
				}
				continue
			}

			fieldname := strings.ToLower(field.Name)
			seen[fieldname] = struct{}{}
			fieldname = safe_keyword(fieldname)

			notag_column(field, fieldname, &columns, m)

			continue
		}

		if mormtag.IsDirective() {
			switch mormtag.tag {
			case IgnoreDirective:
				continue
			case FlattenDirective:
				if field.Type.Kind() != reflect.Struct {
					break
				}

				cols, e := extract_columns(field.Type, m)
				if e != nil {
					return e
				}
				columns = append(columns, cols...)
				continue
			}
		}

		seen[mormtag.fieldname] = struct{}{}
		mormtag.SetFieldName(safe_keyword(mormtag.fieldname))

		// TODO: for more complex types i will need to handle them differenly
		switch field.Type.Kind() {
		case reflect.Array:
			columns = append(columns, mormtag.tag)
		case reflect.Map:
			columns = append(columns, mormtag.tag)
		default:
			columns = append(columns, mormtag.tag)
		}

	}

	query, e := create_table_query(EngineData{table: tablename, columns: strings.Join(columns, ","), dbname: m.databasename, m: m})
	if e != nil {
		return e
	}

	if !m.connected {
		e := m.connect()
		if e != nil {
			panic(e)
		}
	}

	_queryHistory = append(_queryHistory, query)
	_, e = m.db.Exec(query)
	if e != nil {
		panic(e)
	}

	if createdepth == 1 {
		seen = make(map[string]struct{})
	}

	createdepth--
	return nil
}

// Insert creates a new record
func (m *MORM) Insert(model any) error {
	return insert(model, "", m)
}

// InsertByName creates a new record where the tablename is explicit not implicit
func (m *MORM) InsertByName(model any, tablename string) error {
	if tablename == "" {
		return errors.New("tablename is <nil>")
	}
	return insert(model, tablename, m)
}

// Update makes changes to specify fields in the database
func (m *MORM) Update(model any, filters *Filter, fields ...string) Result {
	t := pulltype(model)
	v := pullvalue(model)
	var e error

	query := fmt.Sprintf("update %ss\nset", strings.ToLower(t.Name()))

	var fieldsandvalues []string
	for _, field := range fields {
		fvalue := v.FieldByName(field)

		f, ok := t.FieldByName(field)
		if !ok {
			e = fmt.Errorf("%s field not found", field)
			break
		}

		mtag := gettag(f)

		// TODO: needs a nil check for map, chan pointers and slices

		val, err := tostring(fvalue, f.Type, mtag)
		if err != nil {
			e = err
			break
		}

		fieldsandvalues = append(fieldsandvalues, fmt.Sprintf("%s=%s", mtag.fieldname, val))
	}

	if e != nil {
		return new_result(e, 0)
	}

	query = fmt.Sprintf("%s %s", query, strings.Join(fieldsandvalues, ","))

	if filters != nil {
		wsql, e := filters.WhereSQL()
		if e != nil {
			return error_result(e)
		}
		query += "\n" + wsql
	}

	query += ";"

	_queryHistory = append(_queryHistory, query)

	if !m.connected {
		e := m.connect()
		// NOTE: maybe i don't crash here and try to recover
		Assert(e == nil, e)
	}

	switch m.engine {
	case SQLServer:
		usedb, e := mssql_use_db(m)
		Assert(e == nil, e)
		query = usedb + query
	}

	rslt, e := m.db.Exec(query)

	if e != nil {
		return new_result(e, 0)
	}

	affected, e := rslt.RowsAffected()

	return new_result(e, affected)
}

// Exec executres arbitrary query using the underlying driver
func (m *MORM) Exec(query string, params ...any) (sql.Result, error) {
	if !m.connected {
		e := m.connect()
		Assert(e == nil, e)
	}
	return m.db.Exec(query, params...)
}

func (m *MORM) Query(query string, params ...any) (*sql.Rows, error) {
	Assert(m != nil, "morm instance not initiated")

	if !m.connected {
		e := m.connect()
		if e != nil {
			return nil, e
		}
	}

	return m.db.Query(query, params...)
}

func (m *MORM) QueryRow(query string, params ...any) (*sql.Row, error) {
	Assert(m != nil, "morm instance not initiated")

	if !m.connected {
		e := m.connect()
		if e != nil {
			return nil, e
		}
	}

	return m.db.QueryRow(query, params...), nil
}

func (m *MORM) Drop(model any) error {
	t := pulltype(model)
	return drop(t.Name()+"s", m)
}

func (m *MORM) DropByName(tablename string) error {
	return drop(tablename, m)
}

// DeleteByName deletes records from a tablename based on filter
func (m *MORM) DeleteByName(tablename string, filters *Filter) Result {
	return delete(tablename, filters, m)
}

// Delete deletes a record from the table representation of the model passed in
func (m *MORM) Delete(model any, filters *Filter) Result {
	t := pulltype(model)
	tablename := strings.ToLower(t.Name()) + "s"
	return delete(tablename, filters, m)
}

// Info returns server information
func (m *MORM) Info() (DBInfo, error) {

	switch m.engine {
	case SQLServer:
		info, e := mssql_server_info(m)
		if e != nil {
			return nil, e
		}
		return info, nil
	default:
		return nil, fmt.Errorf("engine %s is not supported for server info", m.engine)
	}

}

// Read read data into the model
func (m *MORM) Read(model any, filters *Filter, tablename string) error {
	return read(model, filters, m, tablename)
}
