package morm

import (
	"fmt"
	"testing"
	"time"

	. "github.com/chapgx/assert"
)

type user struct {
	ID        string `morm:"id text"`
	FirstName string `morm:"first_name text"`
	LastName  string
	Alias     *string `morm:"alias text null"`
	Email     email
	Phone     phone     `morm:":flatten"`
	Now       time.Time `morm:"ts datetime"`
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
		// alias := "mmm"
		u := user{ID: "00", FirstName: "Richard", LastName: "Chapman", Now: time.Now()}
		p := phone{Id: 1, Primary: true, Number: "999999999"}
		u.Phone = p

		e := Insert(&u)
		AssertT(t, e == nil, e)
	})
}

func TestUpdate(t *testing.T) {
	_, e := New(SQLITE, "db.db")
	AssertT(t, e == nil, e)

	alias := "The Big Boss"
	u := user{ID: "007", FirstName: "Richard", LastName: "Bolanos", Alias: &alias}

	filter := NewFilter()
	filter.
		And("id", EQUAL, "01").
		Or("first_name", EQUAL, "Albert")

	g := filter.Group()
	g.
		And("lastname", EQUAL, "Bolanos").
		Or("phone_primary", GREATER, 1).
		AndIsNull("alias")

	result := Update(&u, &filter, "LastName")
	AssertT(t, result.Error == nil, result.Error)

	fmt.Println("rows affected", result.RowsAffected)

	PrintQueryHistory()
}

func TestDelete(t *testing.T) {
	_, e := New(SQLITE, "db.db")
	AssertT(t, e == nil, e)

	filters := NewFilter()
	filters.
		And("id", EQUAL, "00").
		AndIsNull("alias")

	rslt := Delete(user{}, &filters)
	AssertT(t, rslt.Error == nil, rslt.Error)
	PrintQueryHistory()
}
