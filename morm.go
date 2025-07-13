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
	SQLITE = iota + 1
	SQLServer
	POSTGRESS
	MySQL
)

// NOTE: for dev only
var _queryHistory []string

func PrintQueryHistory() {
	for _, query := range _queryHistory {
		fmt.Println(query)
	}
}

var (
	minimalorm *MORM
	seen       = make(map[string]bool)
	// keeps track of nested tables creation execution
	createdepth = 0
)

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
	case SQLITE:
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
	createdepth++

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

			fieldname := strings.ToLower(field.Name)
			seen[fieldname] = true
			fieldname = check_keyword(fieldname)

			columns = append(columns, fmt.Sprintf("%s TEXT", fieldname))
			continue
		}

		if mormtag[0] == ':' {
			// TODO: this is a command do something outside of the norm.
			// examples are :ignore :flatten :new_table and so on

			switch mormtag {
			case ":ignore":
				continue
			case ":flatten":
				if field.Type.Kind() != reflect.Struct {
					break
				}
				cols, e := extract_columns(field.Type)
				if e != nil {
					return e
				}
				columns = append(columns, cols...)
				continue
			}
		}

		split := strings.Split(mormtag, " ")
		seen[split[0]] = true
		split[0] = check_keyword(split[0])
		mormtag = strings.Join(split, " ")

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

	if !minimalorm.connected {
		panic("database is not connected")
	}

	_queryHistory = append(_queryHistory, query)
	_, e := minimalorm.db.Exec(query)
	if e != nil {
		panic(e)
	}

	if createdepth == 1 {
		seen = make(map[string]bool)
	}

	createdepth--
	return nil
}

func extract_columns(model any) ([]string, error) {
	var t reflect.Type
	tt, ok := model.(reflect.Type)
	if ok {
		t = tt
	} else {
		t = reflect.TypeOf(model)
	}

	var columns []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		mormtag := field.Tag.Get("morm")

		// note: untagged are added as text with their field name
		if mormtag == "" {
			fieldname := strings.ToLower(field.Name)
			_, exists := seen[fieldname]
			if exists {
				fieldname = fmt.Sprintf("%s_%s", strings.ToLower(t.Name()), fieldname)
				seen[fieldname] = true
			} else {
				seen[fieldname] = true
			}

			fieldname = check_keyword(fieldname)

			columns = append(columns, fmt.Sprintf("%s TEXT", fieldname))
			continue
		}

		if mormtag[0] == ':' {
			// TODO: this is a command do something outside of the norm.
			// examples are :ignore :flatten :new_table and so on

			switch mormtag {
			case ":ignore":
				continue
			case ":flatten":
				if field.Type.Kind() != reflect.Struct {
					break
				}
				cols, e := extract_columns(field.Type)
				if e != nil {
					return nil, e
				}
				columns = append(columns, cols...)
				mormtag = ""
			}
		}

		// note: checking if the field name has been seen in an upper structure and if it has not recorded
		split := strings.Split(mormtag, " ")
		_, exists := seen[split[0]]
		if exists {
			split[0] = fmt.Sprintf("%s_%s", strings.ToLower(t.Name()), split[0])
			seen[split[0]] = true
		} else {
			seen[split[0]] = true
		}

		split[0] = check_keyword(split[0])
		mormtag = strings.Join(split, " ")

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

	return columns, nil
}
