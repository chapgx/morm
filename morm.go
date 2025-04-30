package morm

import (
	"database/sql"
	"errors"
	"reflect"
	"strings"
)

const (
	SqLiteEngine = iota + 1
	SqlServerEngine
	PostgressEngine
	MySqlEngine
)

var minimalorm *MORM

var (
	ErrMormIsNil = errors.New("err morm is nil")
	ErrDbIsNil   = errors.New("err db interface is nil")
)

type MORM struct {
	db     *sql.DB
	engine int
}

func New(engine int, db *sql.DB) *MORM {
	minimalorm = &MORM{db, engine}
	return minimalorm
}

func (m MORM) Close() error {
	if minimalorm == nil {
		return ErrMormIsNil
	}

	if minimalorm.db == nil {
		return ErrDbIsNil
	}

	return m.db.Close()
}

func CreateTable(model any, tablename string) *MORM {
	var t reflect.Type
	tt, ok := model.(reflect.Type)
	if ok {
		t = tt
	} else {
		t = reflect.TypeOf(model)
	}

	if tablename == "" {
		tablename = strings.ToLower(t.Name())
		if !strings.HasSuffix(tablename, "s") {
			tablename += "s"
		}
	}

	//TODO: change once logic is completed
	return nil
}
