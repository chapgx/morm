package morm

import (
	"testing"

	. "github.com/chapgx/assert"
)

type user struct {
	ID        string `morm:"id text"`
	FirstName string `morm:"firstname text"`
	LastName  string
	Email     email
}

type email struct {
	ID      int `morm:"id int"`
	Address string
}

func TestCore(t *testing.T) {
	_, e := New(SqLiteEngine, "db.db")
	AssertT(t, e == nil, "error was not <nil> ")

	t.Run("create table", func(t *testing.T) {
		var u user
		e := CreateTable(u, "")
		AssertT(t, e == nil, "expected error to be <nil>")
	})
}
