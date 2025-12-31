package morm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	. "github.com/chapgx/assert/v2"
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
	ErrDefaultClientIsNil = errors.New("err defautl client is nil")
	ErrDBIsNil            = errors.New("err db interface is nil")
)

type FnConnect func() error

type MORM struct {
	db        *sql.DB
	connected bool
	engine    int
	connect   FnConnect
}

// SetDefaultClient sets the package level [MORM]
func SetDefaultClient(m MORM) {
	_morm = &m
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
func New(engine int, connectionString string) (MORM, error) {
	m := MORM{engine: engine}
	var e error

	switch engine {
	case SQLITE:
		m.connect = func() error {
			var e error
			_morm.db, e = sql.Open("sqlite", connectionString)
			if e == nil {
				_morm.connected = true
				_, e = _morm.db.Exec(`PRAGMA journal_mode=WAL;PRAGMA synchronous=FULL;`)

				if e != nil {
					return e
				}
			}
			return e
		}
	case MySQL:
		m.connect = func() error {
			var e error
			_morm.db, e = sql.Open("mysql", connectionString)
			if e == nil {
				_morm.connected = true
			}
			return e
		}
	}

	return m, e
}

func (m MORM) Close() error {
	if _morm == nil {
		return ErrDefaultClientIsNil
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
			fieldname = safe_keyword(fieldname)

			notag_column(field, fieldname, &columns)

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

				cols, e := extract_columns(field.Type)
				if e != nil {
					return e
				}
				columns = append(columns, cols...)
				continue
			}
		}

		seen[mormtag.fieldname] = true
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
			fieldname := fmt.Sprintf("%s_%s", strings.ToLower(t.Name()), strings.ToLower(field.Name))
			fieldname = seen_before(fieldname, t.Name())
			notag_column(field, fieldname, &columns)
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
				cols, e := extract_columns(field.Type)
				if e != nil {
					return nil, e
				}
				columns = append(columns, cols...)
				mormtag.tag = ""
			}
		}

		// note: checking if the field name has been seen in an upper structure and if it has not recorded
		mormtag.SetFieldName(fmt.Sprintf("%s_%s", t.Name(), mormtag.fieldname))
		mormtag.SetFieldName(seen_before(mormtag.fieldname, t.Name()))

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

// pullvalue returns the reflect.Value of an any type
func pullvalue(model any) reflect.Value {
	v := reflect.ValueOf(model)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	return v
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
	return insert(model, "")
}

// InsertByName creates a new record where the tablename is explicit not implicit
func InsertByName(model any, tablename string) error {
	if tablename == "" {
		return errors.New("tablename is <nil>")
	}
	return insert(model, tablename)
}

func insert(model any, tblname string) error {
	queries := insertquery(model, true, tblname)
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

// insertquery composes an insert query
func insertquery(model any, independentTable bool, tablename string) []string {
	insertdepth++

	t := pulltype(model)
	v := reflect.ValueOf(model)

	if v.Kind() == reflect.Pointer {
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

		//HACK: this is a hack for now I will need a better solution in the future
		if strings.Contains(mormtag.tag, "autoincrement") {
			continue
		}

		mormtag.SetFieldName(safe_keyword(mormtag.fieldname))
		mormtag.SetFieldName(seen_before(mormtag.fieldname, t.Name()))

		fieldvalue, e := tostring(v.Field(i), field.Type, mormtag)
		Assert(e == nil, e)

		insertline = append(insertline, mormtag.fieldname)
		valuesline = append(valuesline, fieldvalue)
	}

	if tablename == "" {
		tablename = t.Name() + "s"
	}

	qi := fmt.Sprintf("insert into %s(%s)\n", tablename, strings.Join(insertline, ", "))
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
	v := pullvalue(model)

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

		//HACK: this is a hack for now I will need a better solution in the future
		if strings.Contains(mormtag.tag, "autoincrement") {
			continue
		}

		_, exists := seenfields[mormtag.fieldname]
		if exists {
			mormtag.SetFieldName(fmt.Sprintf("%s_%s", t.Name(), mormtag.fieldname))
		}
		seenfields[mormtag.fieldname] = true

		mormtag.SetFieldName(safe_keyword(mormtag.fieldname))

		insertfields = append(insertfields, mormtag.fieldname)
		value, e := tostring(v.Field(i), field.Type, mormtag)
		Assert(e == nil, e)
		insertvalues = append(insertvalues, value)
	}

	insertline := fmt.Sprintf("insert into %ss(%s)\nvalues (%s)", t.Name(), strings.Join(insertfields, ", "), strings.Join(insertvalues, ", "))
	queries = append(queries, insertline)

	return queries
}

// tosstring turns any sql valid type into a string to type to format query
func tostring(val reflect.Value, fieldType reflect.Type, tag MormTag) (string, error) {
	var rval string
	inter := val.Interface()

	switch fieldType.Kind() {
	case reflect.String:
		iv, ok := inter.(string)
		if !ok {
			fmt.Printf("%+v\n", val)
			return "", ErrValIsNotExpectedType
		}
		rval = "'" + iv + "'"
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		iv, ok := inter.(int)
		if !ok {
			return "", ErrValIsNotExpectedType
		}
		rval = strconv.Itoa(iv)
	case reflect.Float32:
		iv, ok := inter.(float32)
		if !ok {
			return "", ErrValIsNotExpectedType
		}
		rval = strconv.FormatFloat(float64(iv), 'f', -1, 32)
	case reflect.Float64:
		iv, ok := inter.(float64)
		if !ok {
			return "", ErrValIsNotExpectedType
		}
		rval = strconv.FormatFloat(iv, 'f', -1, 32)
	case reflect.Bool:
		iv, ok := inter.(bool)
		if !ok {
			return "", ErrValIsNotExpectedType
		}
		if iv {
			rval = "1"
		} else {
			rval = "0"
		}
	case reflect.Pointer:
		if val.IsNil() {
			rval = "null"
			break
		}

		t := fieldType.Elem()
		ptrval, e := tostring(val.Elem(), t, tag)
		if e != nil {
			return "", e
		}
		rval = ptrval
	case reflect.Struct:
		if val.Type() == reflect.TypeOf(time.Time{}) {
			t, ok := inter.(time.Time)
			if !ok {
				return "", fmt.Errorf("exppected interface to be of type time when converting to string")
			}

			switch strings.ToLower(tag.fieldtype) {
			case "integer":
				n := t.UnixMilli()
				rval = strconv.Itoa(int(n))
			case "real":
				t = t.UTC()
				const unixEpochJD = 2440587.7
				n := unixEpochJD + float64(t.UnixNano())/86400e9
				rval = strconv.FormatFloat(n, 'f', -1, 64)
			case "text", "date", "datetime":
				rval = fmt.Sprintf("'%s'", t.Format(time.DateTime))
			case "timestamp":
				rval = t.Format(time.TimeOnly)
			}

			return rval, nil
		}

		return "", fmt.Errorf("attempted to convert to string an unsuported type %s", fieldType.Kind().String())
	default:
		return "", errors.New("interface type has not tostring convertion available")
	}

	return rval, nil
}

// seen_before checks if the fieldname being added to the query has been seen before and it alters the fieldname
// by appending the table name to the field name
func seen_before(fieldname string, tablename string) string {
	// NOTE: may need to change to accomodate for new pre table name format in flatter tables
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

			fieldname := fmt.Sprintf("%s_%s", strings.ToLower(t.Name()), strings.ToLower(field.Name))
			fieldname = seen_before(fieldname, t.Name())

			fieldval, e := tostring(v.Field(i), field.Type, mormtag)
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

		mormtag.SetFieldName(fmt.Sprintf("%s_%s", strings.ToLower(t.Name()), mormtag.fieldname))
		mormtag.SetFieldName(seen_before(mormtag.fieldname, t.Name()))

		fieldvalue, e := tostring(v.Field(i), field.Type, mormtag)
		Assert(e == nil, e)

		fields = append(fields, mormtag.fieldname)
		values = append(values, fieldvalue)
	}

	return fields, values
}

// Update makes changes to specify fields in the database
func Update(model any, filters *Filter, fields ...string) Result {
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

	if !_morm.connected {
		e := _morm.connect()
		// NOTE: maybe i don't crash here and try to recover
		Assert(e == nil, e)
	}

	rslt, e := _morm.db.Exec(query)

	if e != nil {
		return new_result(e, 0)
	}

	affected, e := rslt.RowsAffected()

	return new_result(e, affected)
}

// DeleteByName deletes records from a tablename based on filter
func DeleteByName(tablename string, filters *Filter) Result {
	return delete(tablename, filters)
}

// Delete deletes a record from the table representation of the model passed in
func Delete(model any, filters *Filter) Result {
	t := pulltype(model)
	tablename := strings.ToLower(t.Name()) + "s"
	return delete(tablename, filters)
}

func delete(tablename string, filters *Filter) Result {
	if filters == nil {
		return error_result(errors.New("filters are <nil>"))
	}

	wheresql, e := filters.WhereSQL()
	if e != nil {
		return error_result(e)
	}

	query := fmt.Sprintf("delete from %s\n%s", tablename, wheresql)
	_queryHistory = append(_queryHistory, query)

	if !_morm.connected {
		e := _morm.connect()
		Assert(e == nil, e)
	}

	sqlr, e := _morm.db.Exec(query)
	if e != nil {
		return error_result(e)
	}

	affected, e := sqlr.RowsAffected()
	return new_result(e, affected)
}

func Drop(model any) error {
	t := pulltype(model)
	return drop(t.Name() + "s")
}

func DropByName(tablename string) error {
	return drop(tablename)
}

func drop(tblname string) error {
	//TODO: next drop table functionality
	Assert(_morm != nil, "morm instance has not been initialized")

	if !_morm.connected {
		e := _morm.connect()
		return e
	}

	_, e := _morm.db.Exec("drop table " + tblname + ";")

	return e
}

func Query(query string, params ...any) (*sql.Rows, error) {
	Assert(_morm != nil, "morm instance not initiated")

	if !_morm.connected {
		e := _morm.connect()
		if e != nil {
			return nil, e
		}
	}

	return _morm.db.Query(query, params...)
}

func QueryRow(query string, params ...any) (*sql.Row, error) {
	Assert(_morm != nil, "morm instance not initiated")

	if !_morm.connected {
		e := _morm.connect()
		if e != nil {
			return nil, e
		}
	}

	return _morm.db.QueryRow(query, params...), nil
}

// Exec executres arbitrary query using the underlying driver
func Exec(query string, params ...any) (sql.Result, error) {
	if !_morm.connected {
		e := _morm.connect()
		Assert(e == nil, e)
	}
	return _morm.db.Exec(query, params...)
}

// Close closes databse connection
func Close() error {
	Assert(_morm != nil, "morm instance is <nil>")

	if !_morm.connected {
		return errors.New("morm instane is not connected")
	}

	e := _morm.db.Close()

	if e != nil {
		return e
	}

	_morm.connected = false

	return nil
}

func Read(model any, filters *Filter) error { return read(model, filters) }

func read(model any, filters *Filter) error {
	// TODO: next finish read funtion
	_ = pulltype(model)
	_ = pullvalue(model)
	return nil
}
