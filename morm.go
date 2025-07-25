package morm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	. "github.com/chapgx/assert"
	_ "github.com/go-sql-driver/mysql"
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
	_morm *MORM
	seen  = make(map[string]bool)
	// keeps track of nested tables creation execution
	createdepth = 0
	insertdepth = 0
)

var (
	ErrMormIsNil = errors.New("err morm is nil")
	ErrDBIsNil   = errors.New("err db interface is nil")
)

type FnConnect func() error

type MORM struct {
	db        *sql.DB
	connected bool
	engine    int
	connect   FnConnect
}

func New(engine int, connectionString string) (*MORM, error) {
	_morm = &MORM{engine: engine}
	var e error

	switch engine {
	case SQLITE:
		_morm.connect = func() error {
			var e error
			_morm.db, e = sql.Open("sqlite", connectionString)
			if e == nil {
				_morm.connected = true
			}
			return e
		}
	case MySQL:
		_morm.connect = func() error {
			var e error
			_morm.db, e = sql.Open("mysql", connectionString)
			if e == nil {
				_morm.connected = true
			}
			return e
		}
	}

	return _morm, e
}

func (m MORM) Close() error {
	if _morm == nil {
		return ErrMormIsNil
	}

	if _morm.db == nil {
		return ErrDBIsNil
	}

	if !_morm.connected {
		return errors.New("database is not connected")
	}

	return m.db.Close()
}

// CreateTable creates a new table if it does not exists, tablename is used if is not <nil>
// otherwise the struct name is used as the table name.
func CreateTable(model any, tablename string) error {
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

			notag_column(field, fieldname, &columns)

			continue
		}

		if mormtag.IsDirective() {
			// TODO: this is a command do something outside of the norm.
			// examples are :ignore :flatten :new_table and so on

			switch mormtag.tag {
			case IgnoreDirective:
				continue
			case FlattenDirective:
				// TODO: next (when flatting table in creation append all fields with struct name)
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

		seen[mormtag.fieldname] = true
		mormtag.SetFieldName(check_keyword(mormtag.fieldname))

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

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tablename, strings.Join(columns, ","))

	if !_morm.connected {
		e := _morm.connect()
		if e != nil {
			panic(e)
		}
	}

	_queryHistory = append(_queryHistory, query)
	_, e := _morm.db.Exec(query)
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
	t := pulltype(model)

	var columns []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		mormtag := gettag(field)

		// note: untagged are added as text with their field name
		if mormtag.IsEmpty() {
			fieldname := strings.ToLower(field.Name)
			fieldname = seen_before(fieldname, t.Name())
			fieldname = check_keyword(fieldname)
			notag_column(field, fieldname, &columns)
			continue
		}

		if mormtag.IsDirective() {
			// TODO: this is a command do something outside of the norm.
			// examples are :ignore :flatten :new_table and so on

			switch mormtag.tag {
			case IgnoreDirective:
				continue
			case FlattenDirective:
				if field.Type.Kind() != reflect.Struct {
					break
				}
				cols, e := extract_columns(field.Type)
				if e != nil {
					return nil, e
				}
				columns = append(columns, cols...)
				mormtag.tag = ""
			}
		}

		// note: checking if the field name has been seen in an upper structure and if it has not recorded
		mormtag.SetFieldName(seen_before(mormtag.fieldname, t.Name()))
		mormtag.SetFieldName(check_keyword(mormtag.fieldname))

		// TODO: for more complex types i will need to handle them differenly
		switch field.Type.Kind() {
		case reflect.Array:
			columns = append(columns, mormtag.fieldname)
		case reflect.Map:
			columns = append(columns, mormtag.fieldname)
		default:
			columns = append(columns, mormtag.fieldname)
		}

	}

	return columns, nil
}

func pulltype(model any) reflect.Type {
	var t reflect.Type
	tt, ok := model.(reflect.Type)
	if ok {
		return tt
	}

	t = reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}

	return t
}

// notag_column creates the sql syntax with the correct type based on the struct field type
func notag_column(field reflect.StructField, fieldname string, columns *[]string) {
	switch field.Type.Kind() {
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
		*columns = append(*columns, fmt.Sprintf("%s integer", fieldname))
	case reflect.String:
		*columns = append(*columns, fmt.Sprintf("%s text", fieldname))
	case reflect.Bool:
		*columns = append(*columns, fmt.Sprintf("%s integer check (%s IN(0,1))", fieldname, fieldname))
	case reflect.Array:
		if field.Type.Elem().Kind() == reflect.Uint8 {
			*columns = append(*columns, fmt.Sprintf("%s blob", fieldname))
		}
	}
}

// Insert creates a new record
func Insert(model any) error {
	queries := insertquery(model, true)
	Assert(len(queries) >= 1, "expected to have queries to process but found none")

	if !_morm.connected {
		_morm.connect()
	}

	var e error

	for _, q := range queries {
		_queryHistory = append(_queryHistory, queries...)
		_, e = _morm.db.Exec(q)
		if e != nil {
			break
		}
	}

	return e
}

// insertquery composes insert query
func insertquery(model any, independentTable bool) []string {
	insertdepth++

	t := pulltype(model)
	v := reflect.ValueOf(model)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	executionchain := make([]string, 0)

	var insertline []string
	var valuesline []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		mormtag := gettag(field)

		// no tag branch
		if mormtag.IsEmpty() {

			fname, fval, q := emptytagprocess(field, v, t, i, nil)
			if fname == "" && fval == "" {
				executionchain = append(executionchain, q...)
				continue
			}

			insertline = append(insertline, fname)
			valuesline = append(valuesline, fval)
			continue
		}

		// directive branch
		if mormtag.IsDirective() {
			switch mormtag.tag {
			case IgnoreDirective:
				continue
			case FlattenDirective:
				if t.Kind() == reflect.Struct {
					i := v.Field(i).Interface()
					fields, values := pull_fields_and_values(i)
					insertline = append(insertline, fields...)
					valuesline = append(valuesline, values...)
				}
				continue
			}
		}

		mormtag.SetFieldName(check_keyword(mormtag.fieldname))
		mormtag.SetFieldName(seen_before(mormtag.fieldname, t.Name()))

		valueinterface := v.Field(i).Interface()
		fieldvalue, e := tostring(valueinterface, field.Type.Kind())
		Assert(e == nil, e)

		insertline = append(insertline, mormtag.fieldname)
		valuesline = append(valuesline, fieldvalue)
	}

	qi := fmt.Sprintf("insert into %ss(%s)\n", t.Name(), strings.Join(insertline, ", "))
	qv := fmt.Sprintf("values (%s)", strings.Join(valuesline, ", "))
	qi += qv

	executionchain = append(executionchain, qi)

	if insertdepth <= 1 {
		seen = make(map[string]bool)
	}

	insertdepth--
	return executionchain
}

// insert_adjecent composes the insert query for a nester struct adjecent to it's parent struct
func insert_adjecent(model any, seenfields map[string]bool) []string {
	if seenfields == nil {
		seenfields = make(map[string]bool)
	}

	t := pulltype(model)
	v := reflect.ValueOf(model)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var queries []string
	var insertfields []string
	var insertvalues []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		mormtag := gettag(field)

		if mormtag.IsEmpty() {
			fname, fval, qrs := emptytagprocess(field, v, t, i, seenfields)
			if fname == "" && fval == "" {
				queries = append(queries, qrs...)
				continue
			}

			insertfields = append(insertfields, fname)
			insertvalues = append(insertvalues, fval)
			continue
		}

		if mormtag.IsDirective() {
			switch mormtag.tag {
			case IgnoreDirective:
				continue
			case FlattenDirective:
				if field.Type.Kind() == reflect.Struct {
					interfa := v.Field(i).Interface()
					fields, values := pull_fields_and_values(interfa)
					insertfields = append(insertfields, fields...)
					insertvalues = append(insertvalues, values...)
				}
				continue
			}
		}

		_, exists := seenfields[mormtag.fieldname]
		if exists {
			mormtag.SetFieldName(fmt.Sprintf("%s_%s", t.Name(), mormtag.fieldname))
		}
		seenfields[mormtag.fieldname] = true

		mormtag.SetFieldName(check_keyword(mormtag.fieldname))

		insertfields = append(insertfields, mormtag.fieldname)
		value, e := tostring(v.Field(i).Interface(), field.Type.Kind())
		Assert(e == nil, e)
		insertvalues = append(insertvalues, value)
	}

	insertline := fmt.Sprintf("insert into %ss(%s)\nvalues (%s)", t.Name(), strings.Join(insertfields, ", "), strings.Join(insertvalues, ", "))
	queries = append(queries, insertline)

	return queries
}

// tosstring turns any sql valid type into a string to type to format query
func tostring(val interface{}, kind reflect.Kind) (string, error) {
	var rval string
	switch kind {
	case reflect.String:
		iv, ok := val.(string)
		if !ok {
			return "", ErrValIsNotExpectedType
		}
		rval = "'" + iv + "'"
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		iv, ok := val.(int)
		if !ok {
			return "", ErrValIsNotExpectedType
		}
		rval = strconv.Itoa(iv)
	case reflect.Bool:
		iv, ok := val.(bool)
		if !ok {
			return "", ErrValIsNotExpectedType
		}

		if iv {
			rval = "1"
		} else {
			rval = "0"
		}
	default:
		return "", errors.New("interface type has not tostring convertion available")
	}

	return rval, nil
}

// seen_before checks if the filename being added to the query has been seen before and it alters the filename
// by appending the table name to the field name
func seen_before(fieldname string, tablename string) string {
	_, found := seen[fieldname]
	if found {
		fieldname = fmt.Sprintf("%s_%s", tablename, fieldname)
		seen[fieldname] = true
	} else {
		seen[fieldname] = true
	}
	return fieldname
}

// pull_fields_and_values returns fiels and values from a struct
func pull_fields_and_values(model any) (fields []string, values []string) {
	t := pulltype(model)
	v := reflect.ValueOf(model)

	_, ok := model.(reflect.Type)
	if ok {
		fmt.Println("it is a type of reflect.Type")
	}

	if v.Kind() == reflect.Ptr {
		fmt.Println("is pointer seeting elem")
		v = v.Elem()
		fmt.Printf("%+v\n", v)
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		mormtag := gettag(field)

		// no tag branch
		if mormtag.IsEmpty() {

			if field.Type.Kind() == reflect.Struct {
				// TODO: figure out how to handle nested structures
				// NOTE: ignoring this for now
				continue
			}

			fieldname := strings.ToLower(field.Name)
			fieldname = seen_before(fieldname, t.Name())
			fieldname = check_keyword(fieldname)

			fieldvalueI := v.Field(i).Interface()

			fieldval, e := tostring(fieldvalueI, field.Type.Kind())
			Assert(e == nil, e)

			fields = append(fields, fieldname)
			values = append(values, fieldval)
			continue
		}

		// directive branch
		if mormtag.IsDirective() {
			switch mormtag.tag {
			case IgnoreDirective:
				continue
			case FlattenDirective:
				if t.Kind() == reflect.Struct {
					// TODO: figure out how to handle nested structures directives
					continue
				}
				continue
			}
		}

		mormtag.SetFieldName(check_keyword(mormtag.fieldname))
		mormtag.SetFieldName(seen_before(mormtag.fieldname, t.Name()))

		fmt.Println(mormtag.fieldname)

		valueinterface := v.Field(i).Interface()
		fieldvalue, e := tostring(valueinterface, field.Type.Kind())
		Assert(e == nil, e)

		fields = append(fields, mormtag.fieldname)
		values = append(values, fieldvalue)
	}

	return fields, values
}
