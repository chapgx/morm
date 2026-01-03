package test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/chapgx/assert/v2"
	"github.com/chapgx/morm"
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
	sqlite, e := morm.New(morm.SQLITE, "./data/db.db")
	AssertT(t, e == nil, e)

	mssql, e := morm.NewWithName(morm.SQLServer, "sqlserver://sa:c1UhH%5E%25h%25lWPqXS2%233tE@127.0.0.1:1433?database=master", "movies")
	AssertT(t, e == nil, e)

	t.Run("create table", func(t *testing.T) {
		var u user

		e = sqlite.CreateTable(u, "")
		AssertT(t, e == nil, e)
	})

	t.Run("save data", func(t *testing.T) {
		u := user{ID: "00", FirstName: "Richard", LastName: "Chapman", Now: time.Now()}
		p := phone{Id: 1, Primary: true, Number: "999999999"}
		u.Phone = p

		e = sqlite.Insert(&u)
		AssertT(t, e == nil, e)
		morm.PrintQueryHistory()
	})

	t.Run("read_data", func(t *testing.T) {
		var u user
		var msu mssql_user

		e := sqlite.Read(&u, nil, "")
		AssertT(t, e == nil, e)

		e = mssql.Read(&msu, nil, "movies")
		AssertT(t, e == nil, e)

	})
}

func TestUpdate(t *testing.T) {
	e := morm.SetDefaultClient(morm.SQLITE, "./data/db.db")
	AssertT(t, e == nil, e)

	orm, e := morm.New(morm.SQLITE, "./data/independent.db")
	AssertT(t, e == nil, e)

	alias := "The Big Boss"
	u := user{ID: "01", FirstName: "Richard", LastName: "Bolanos", Alias: &alias}

	filter := morm.NewFilter()
	filter.
		And("id", morm.EQUAL, "00").
		Or("first_name", morm.EQUAL, "Albert")

	g := filter.Group()
	g.
		And("lastname", morm.EQUAL, "Bolanos").
		Or("phone_primary", morm.GREATER, 1).
		AndIsNull("alias")

	result := morm.Update(&u, &filter, "LastName")
	AssertT(t, result.Error == nil, result.Error)
	fmt.Println("rows affected", result.RowsAffected)

	result = orm.Update(&u, &filter, "LastName")
	AssertT(t, result.Error == nil, result.Error)
	fmt.Println("rows affected", result.RowsAffected)

	morm.PrintQueryHistory()
}

func TestDelete(t *testing.T) {
	e := morm.SetDefaultClient(morm.SQLITE, "./data/db.db")
	AssertT(t, e == nil, e)
	orm, e := morm.New(morm.SQLITE, "./data/independent.db")
	AssertT(t, e == nil, e)

	filters := morm.NewFilter()
	filters.
		And("id", morm.EQUAL, "00").
		AndIsNull("alias")

	rslt := morm.Delete(user{}, &filters)
	AssertT(t, rslt.Error == nil, rslt.Error)
	rslt = orm.Delete(user{}, &filters)
	AssertT(t, rslt.Error == nil, rslt.Error)

	morm.PrintQueryHistory()
}
