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
	Phone     phone `morm:":flatten"`
}

type email struct {
	ID      int `morm:"id int"`
	Address string
}

type phone struct {
	Id      int
	Primary bool
	Number  string
}

func TestCore(t *testing.T) {
	_, e := New(SQLITE, "db.db")
	AssertT(t, e == nil, "error was not <nil> ")

	t.Run("create table", func(t *testing.T) {
		var u user
		e := CreateTable(u, "")
		AssertT(t, e == nil, "expected error to be <nil>")

		// PrintQueryHistory()
	})

	t.Run("save data", func(t *testing.T) {
		u := user{ID: "00", FirstName: "Richard", LastName: "Chapman"}
		p := phone{Id: 1, Primary: true, Number: "999999999"}
		u.Phone = p

		e := Insert(&u)
		AssertT(t, e == nil, e)
	})
}
