package morm

import (
	"fmt"
	"reflect"
	"strings"

	. "github.com/chapgx/assert"
)

// MormTag is a representation of the morm notation tag
type MormTag struct {
	tag       string
	split     []string
	fieldname string
}

// IsEmpty checks if the morm tag is an empty tag
func (mt MormTag) IsEmpty() bool {
	return mt.tag == ""
}

// IsDirective checks if the notation tag is a morm directive
func (mt MormTag) IsDirective() bool {
	return mt.tag[0] == ':'
}

func (mt *MormTag) SetFieldName(fn string) {
	mt.fieldname = fn
	mt.split[0] = fn
	mt.tag = strings.Join(mt.split, " ")
}

func gettag(field reflect.StructField) MormTag {
	mormtag := field.Tag.Get("morm")
	mt := MormTag{tag: mormtag}

	if mt.IsEmpty() {
		return mt
	}

	if !mt.IsDirective() {
		mt.split = strings.Split(mt.tag, " ")
		mt.fieldname = mt.split[0]
	}

	return mt
}

func emptytagprocess(field reflect.StructField, v reflect.Value, t reflect.Type, index int, seenfields map[string]bool) (fieldname, fieldval string, query []string) {
	if field.Type.Kind() == reflect.Struct {
		iface := v.Field(index).Interface()
		query = insert_adjecent(iface, nil)
		return "", "", query
	}

	if seenfields == nil {
		seenfields = make(map[string]bool)
	}

	fieldname = strings.ToLower(field.Name)

	_, exists := seenfields[fieldname]
	if exists {
		fieldname = fmt.Sprintf("%s_%s", t.Name(), fieldname)
	}
	seenfields[fieldname] = true

	fieldname = check_keyword(fieldname)

	fieldvalueI := v.Field(index).Interface()
	fieldval, e := tostring(fieldvalueI, field.Type.Kind())
	Assert(e == nil, e)

	return fieldname, fieldval, nil
}
