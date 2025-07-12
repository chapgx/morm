package morm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	_ "modernc.org/sqlite"
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
	ErrDBIsNil   = errors.New("err db interface is nil")
)

type MORM struct {
	db        *sql.DB
	connected bool
	engine    int
}

func New(engine int, connectionString string) (*MORM, error) {
	minimalorm = &MORM{engine: engine}
	var e error

	switch engine {
	case SqLiteEngine:
		minimalorm.db, e = sql.Open("sqlite", connectionString)
	}

	if e != nil {
		return nil, e
	}

	minimalorm.connected = true

	return minimalorm, e
}

func (m MORM) Close() error {
	if minimalorm == nil {
		return ErrMormIsNil
	}

	if minimalorm.db == nil {
		return ErrDBIsNil
	}

	return m.db.Close()
}

// CreateTable creates a new table if it does not exists, tablename is used if is not <nil>
// otherwise the struct name is used as the table name.
func CreateTable(model any, tablename string) error {
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

	var columns []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		mormtag := field.Tag.Get("morm")

		// note: untagged are added as text with their field name
		if mormtag == "" {
			if field.Type.Kind() == reflect.Struct {
				e := CreateTable(field.Type, "")
				if e != nil {
					panic(e)
				}
				continue
			}
			columns = append(columns, fmt.Sprintf("%s TEXT", strings.ToLower(field.Name)))
			continue
		}

		if mormtag[0] == ':' {
			// TODO: this is a command do something outside of the norm.
			// examples are :ignore :flatten :new_table and so on

			switch mormtag {
			case ":ignore":
				continue
			}
		}

		// TODO: for more complex types i will need to handle them differenly
		switch field.Type.Kind() {
		case reflect.Array:
			columns = append(columns, mormtag)
		case reflect.Map:
			columns = append(columns, mormtag)
		default:
			columns = append(columns, mormtag)
		}

	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tablename, strings.Join(columns, ","))

	fmt.Println(query)

	if !minimalorm.connected {
		panic("database is not connected")
	}

	_, e := minimalorm.db.Exec(query)
	if e != nil {
		panic(e)
	}

	return nil
}
