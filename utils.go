package morm

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"
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
	return t == reflect.TypeOf(time.Time{})
}
