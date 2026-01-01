package morm

import (
	"database/sql"
	"errors"
	. "github.com/chapgx/assert/v2"
	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

var (
	_morm *MORM
	seen  = make(map[string]bool)

	// keeps track of nested tables creation execution
	createdepth = 0
	insertdepth = 0
)

var (
	ErrDefaultClientIsNil = errors.New("err defautl client is nil")
	ErrDBIsNil            = errors.New("err db interface is nil")
)

// SetDefaultClient sets the package level [MORM]
func SetDefaultClient(engine int, connString string) error {
	m, e := New(engine, connString)
	_morm = m
	return e
}

// GetDefault returns the package level [MORM] or an error
func GetDefault() (*MORM, error) {
	if _morm == nil {
		return nil, ErrDefaultClientIsNil
	}
	return _morm, nil
}

// GetDefaultMust returns default [MORM] panics on error
func GetDefaultMust() *MORM {
	return Must(GetDefault())
}

// New creates and return a new [MORM] based on the engine and connectionString
func New(engine int, connectionString string) (*MORM, error) {
	m := MORM{engine: engine}
	var e error

	switch engine {
	case SQLITE:
		m.connect = func() error {
			var e error
			m.db, e = sql.Open("sqlite", connectionString)
			if e == nil {
				m.connected = true
				_, e = m.db.Exec(`PRAGMA journal_mode=WAL;PRAGMA synchronous=FULL;`)
				if e != nil {
					return e
				}
			}
			return e
		}
	case MySQL:
		m.connect = func() error {
			var e error
			m.db, e = sql.Open("mysql", connectionString)
			if e == nil {
				m.connected = true
			}
			return e
		}
	}

	return &m, e
}

// CreateTable creates a new table if it does not exists, tablename is used if is not <nil>
// otherwise the struct name is used as the table name.
func CreateTable(model any, tablename string) error {
	return _morm.CreateTable(model, tablename)
}

// Insert creates a new record
func Insert(model any) error {
	return insert(model, "", _morm)
}

// InsertByName creates a new record where the tablename is explicit not implicit
func InsertByName(model any, tablename string) error {
	if tablename == "" {
		return errors.New("tablename is <nil>")
	}
	return insert(model, tablename, _morm)
}

// Update makes changes to specify fields in the database
func Update(model any, filters *Filter, fields ...string) Result {
	return _morm.Update(model, filters, fields...)
}

// Exec executres arbitrary query using the underlying driver
func Exec(query string, params ...any) (sql.Result, error) {
	return _morm.Exec(query, params...)
}

// Close closes databse connection for the default [MORM] client
func Close() error {
	return _morm.Close()
}

func Query(query string, params ...any) (*sql.Rows, error) {
	return _morm.Query(query, params...)
}

func QueryRow(query string, params ...any) (*sql.Row, error) {
	return _morm.QueryRow(query, params...)
}

func Drop(model any) error {
	return _morm.Drop(model)
}

func DropByName(tablename string) error {
	return DropByName(tablename)
}

// DeleteByName deletes records from a tablename based on filter
func DeleteByName(tablename string, filters *Filter) Result {
	return _morm.DeleteByName(tablename, filters)
}

// Delete deletes a record from the table representation of the model passed in
func Delete(model any, filters *Filter) Result {
	return _morm.Delete(model, filters)
}

func Read(model any, filters *Filter) error { return _morm.Read(model, filters) }
