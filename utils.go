package morm

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	. "github.com/chapgx/assert/v2"
)

// toint turns any integer type to int
func toint64(i any) (int64, error) {
	v, ok := i.(int)
	if ok {
		return int64(v), nil
	}

	v8, ok := i.(int8)
	if ok {
		return int64(v8), nil
	}

	v16, ok := i.(int16)
	if ok {
		return int64(v16), nil
	}

	v32, ok := i.(int32)
	if ok {
		return int64(v32), nil
	}

	v64, ok := i.(int64)
	if ok {
		return int64(v64), nil
	}

	return 0, errors.New("no convertion candidate found")
}

// touint turns any unsigned integer type to uint
func touint64(i any) (uint64, error) {
	v, ok := i.(uint)
	if ok {
		return uint64(v), nil
	}

	v8, ok := i.(uint8)
	if ok {
		return uint64(v8), nil
	}

	v16, ok := i.(uint16)
	if ok {
		return uint64(v16), nil
	}

	v32, ok := i.(uint32)
	if ok {
		return uint64(v32), nil
	}

	v64, ok := i.(uint64)
	if ok {
		return uint64(v64), nil
	}

	return 0, errors.New("no convertion candidate found")
}

// anytostr tranforms common types to string representation for sql
func anytostr(val any) (string, error) {
	if val == nil {
		return "null", nil
	}

	t := pulltype(val)
	var stringval string

	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, e := toint64(val)
		if e != nil {
			return "", e
		}
		sv := strconv.Itoa(int(v))
		stringval = sv
	case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint16, reflect.Uint64:
		v, e := touint64(val)
		if e != nil {
			return "", e
		}
		sv := strconv.Itoa(int(v))
		stringval = sv
	case reflect.String:
		sv, ok := val.(string)
		if !ok {
			return "", fmt.Errorf("%v unable to convert to string", val)
		}
		stringval = fmt.Sprintf("'%s'", sv)
	case reflect.Bool:
		boolrlst, ok := val.(bool)
		if !ok {
			return "", fmt.Errorf("%v unable to convert to boolean", val)
		}
		if boolrlst {
			stringval = "1"
		} else {
			stringval = "0"
		}
	case reflect.Struct:
		if istimetype(t) {
			tval, ok := val.(time.Time)
			if !ok {
				return "", fmt.Errorf("%v unable to convert to time.Time", tval)
			}
			stringval = tval.Format(time.RFC3339)
			break
		}

		return "", fmt.Errorf("%v struct type is not supported", val)
	default:
		return "", fmt.Errorf("%v kind is not supported", val)
	}

	return stringval, nil
}

// istimetype compared a tyme to a time.Time struct
func istimetype(t reflect.Type) bool {
	return t == reflect.TypeFor[time.Time]()
}

// pull_fields_and_values returns fiels and values from a struct
func pull_fields_and_values(model any) (fields []string, values []string) {
	t := pulltype(model)
	v := reflect.ValueOf(model)

	_, ok := model.(reflect.Type)
	if ok {
		fmt.Println("it is a type of reflect.Type")
	}

	if v.Kind() == reflect.Pointer {
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

func delete(tablename string, filters *Filter, m *MORM) Result {
	if filters == nil {
		return error_result(errors.New("filters are <nil>"))
	}

	wheresql, e := filters.WhereSQL()
	if e != nil {
		return error_result(e)
	}

	query := fmt.Sprintf("delete from %s\n%s", tablename, wheresql)
	_queryHistory = append(_queryHistory, query)

	if !m.connected {
		e := m.connect()
		Assert(e == nil, e)
	}

	sqlr, e := m.db.Exec(query)
	if e != nil {
		return error_result(e)
	}

	affected, e := sqlr.RowsAffected()
	return new_result(e, affected)
}

func drop(tblname string, m *MORM) error {
	//TODO: next drop table functionality
	Assert(m != nil, "morm instance has not been initialized")

	if !m.connected {
		e := m.connect()
		return e
	}

	_, e := m.db.Exec("drop table " + tblname + ";")

	return e
}

func select_query(model reflect.Type, filters *Filter, m *MORM, tablename string, is_container bool) (string, error) {

	var limited_selection string
	if !is_container {
		switch m.engine {
		case SQLITE:
			limited_selection = "LIMIT 1"
		case SQLServer:
			limited_selection = "top(1)"
		}
	}

	var selected_fields []string
	for i := 0; i < model.NumField(); i += 1 {
		field := model.Field(i)
		tag := gettag(field)

		if tag.IsEmpty() {
			if field.Type.Kind() == reflect.Struct {
				continue
			}

			selected_fields = append(selected_fields, strings.ToLower(field.Name))
			continue
		}

		if tag.IsDirective() {
			switch tag.tag {
			case IgnoreDirective:
				continue
			case FlattenDirective:
				//TODO: we need to cash this so we can run it later and add it to the data itself. Or we ignore it and let the data addition add it
				continue
			}
		}

		tag.fieldname = safe_keyword(tag.fieldname)
		selected_fields = append(selected_fields, tag.fieldname)

	}

	if tablename == "" {
		tablename = model.Name()
	}

	var where_clause string
	if filters != nil {
		q, e := filters.WhereSQL()
		if e != nil {
			return "", e
		}
		where_clause = q
	}

	var query string
	switch m.engine {
	case SQLITE:
		if where_clause != "" {
			query = fmt.Sprintf("select %s\nfrom %s\n%s\n%s;", strings.Join(selected_fields, ", "), tablename, where_clause, limited_selection)
			break
		}
		query = fmt.Sprintf("select %s\nfrom %s\n%s;", strings.Join(selected_fields, ", "), tablename, limited_selection)
	case SQLServer:
		if where_clause != "" {
			query = fmt.Sprintf("select %s %s\nfrom %s\n%s;", limited_selection, strings.Join(selected_fields, ", "), tablename, where_clause)
			break
		}
		query = fmt.Sprintf("select %s %s\nfrom %s;", limited_selection, strings.Join(selected_fields, ", "), tablename)
	}

	return query, nil
}

func read(model any, filters *Filter, m *MORM, tablename string) error {
	is_container := false
	t := pulltype(model)
	// v := pullvalue(model)

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = t.Elem()
	}

	query, e := select_query(t, filters, m, tablename, is_container)
	if e != nil {
		return e
	}

	//TODO: HIGH PRIORITY! finish logic

	fmt.Println("from engine", m.engine)
	fmt.Println(query)
	fmt.Println()

	return nil
}

// seen_before checks if the fieldname being added to the query has been seen before and it alters the fieldname
// by appending the table name to the field name
func seen_before(fieldname string, tablename string) string {
	// NOTE: may need to change to accomodate for new pre table name format in flatter tables
	_, found := seen[fieldname]
	if found {
		fieldname = fmt.Sprintf("%s_%s", tablename, fieldname)
		seen[fieldname] = struct{}{}
	} else {
		seen[fieldname] = struct{}{}
	}
	return fieldname
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

func extract_columns(model any, m *MORM) ([]string, error) {
	t := pulltype(model)

	var columns []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		mormtag := gettag(field)

		// note: untagged are added as text with their field name
		if mormtag.IsEmpty() {
			fieldname := fmt.Sprintf("%s_%s", strings.ToLower(t.Name()), strings.ToLower(field.Name))
			fieldname = seen_before(fieldname, t.Name())
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
	if t.Kind() == reflect.Pointer {
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
func notag_column(field reflect.StructField, fieldname string, columns *[]string, m *MORM) {
	switch field.Type.Kind() {
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
		*columns = append(*columns, fmt.Sprintf("%s integer", fieldname))
	case reflect.String:
		var declaration string
		switch m.engine {
		case SQLITE:
			declaration = fmt.Sprintf("%s text", fieldname)
		case SQLServer:
			declaration = fmt.Sprintf("%s varchar(max)", fieldname)
		default:
			panic(fmt.Sprintf("engine %s not supported for notag string field declaration", m.engine))
		}
		*columns = append(*columns, declaration)
	case reflect.Bool:
		*columns = append(*columns, fmt.Sprintf("%s integer check (%s IN(0,1))", fieldname, fieldname))
	case reflect.Array:
		if field.Type.Elem().Kind() == reflect.Uint8 {
			*columns = append(*columns, fmt.Sprintf("%s blob", fieldname))
		}
	case reflect.Slice:
		fmt.Println("made it so slicees ignoring")
	}
}

func insert(model any, tblname string, m *MORM) error {
	var queries []string
	insert_queries := insertquery(model, true, tblname, m)
	Assert(len(insert_queries) >= 1, "expected to have queries to process but found none")

	switch m.engine {
	case SQLServer:
		usedb, e := mssql_use_db(m)
		Assert(e == nil, e)
		queries = append(queries, usedb[:len(usedb)-1])
		queries = append(queries, insert_queries...)
	case SQLITE:
		queries = insert_queries
	}

	if !m.connected {
		m.connect()
	}

	query := strings.Join(queries, ";\n\n")
	_queryHistory = append(_queryHistory, query)

	_, e := m.db.Exec(query)
	if e != nil {
		return e
	}

	return e
}

// insertquery composes an insert query
func insertquery(model any, independentTable bool, tablename string, m *MORM) []string {
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
		seen = make(map[string]struct{})
	}

	insertdepth--
	return executionchain
}
