package morm

import (
	"errors"
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
