package morm

import (
	"fmt"
	"reflect"
	"strings"

	. "github.com/chapgx/assert/v2"
)

// MormTag is a representation of the morm notation tag
type MormTag struct {
	tag       string
	fieldtype string
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
		mt.fieldname = strings.ToLower(field.Name)
		return mt
	}

	if !mt.IsDirective() {
		mt.split = strings.Split(mt.tag, " ")
		mt.fieldname = mt.split[0]
		mt.fieldtype = mt.split[1]
	}

	return mt
}

func emptytagprocess(field reflect.StructField, v reflect.Value, t reflect.Type, index int, seenfields map[string]bool) (name, value string, query []string) {

	// struct control structure
	if field.Type.Kind() == reflect.Struct {
		iface := v.Field(index).Interface()
		query = insert_adjecent(iface, nil)
		return "", "", query
	}

	if seenfields == nil {
		seenfields = make(map[string]bool)
	}

	name = strings.ToLower(field.Name)

	_, exists := seenfields[name]
	if exists {
		name = fmt.Sprintf("%s_%s", t.Name(), name)
	}
	seenfields[name] = true

	name = safe_keyword(name)

	fieldvalue := v.Field(index)
	value, e := tostring(fieldvalue, field.Type, MormTag{})
	Assert(e == nil, e)

	return name, value, nil
}
