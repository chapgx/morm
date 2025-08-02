package morm

import (
	"fmt"
	"testing"

	. "github.com/chapgx/assert"
)

type user struct {
	ID        string `morm:"id text"`
	FirstName string `morm:"first_name text"`
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

func TestUpdate(t *testing.T) {
	_, e := New(SQLITE, "db.db")
	AssertT(t, e == nil, e)

	u := user{ID: "007", FirstName: "Richard", LastName: "Bolanos"}

	filter := NewFilter()
	filter.
		And("id", "01").
		Or("first_name", "Albert")

	g := filter.Group()
	g.
		And("lastname", "Bolanos").
		Or("phone_primary", 1)

	result := Update(&u, &filter, []string{"LastName"})
	AssertT(t, result.Error == nil, result.Error)

	fmt.Println("rows affected", result.RowsAffected)

	PrintQueryHistory()
}

func TestDelete(t *testing.T) {
	_, e := New(SQLITE, "db.db")
	AssertT(t, e == nil, e)

	rslt := Delete("users", map[string]any{"id": "00"})
	AssertT(t, rslt.Error == nil, rslt.Error)
	PrintQueryHistory()
}
