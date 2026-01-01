package morm

import (
	"database/sql"
	"errors"
	. "github.com/chapgx/assert/v2"
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
	if _morm == nil {
		return ErrDefaultClientIsNil
	}
	return _morm.CreateTable(model, tablename)
}
