package test

import (
	"testing"

	. "github.com/chapgx/assert/v2"
	"github.com/chapgx/morm"
)

type mssql_user struct {
	FirstName string `morm:"firstname varchar(150) not null"`
}

func TestMSSQLCore(t *testing.T) {
	orm, e := morm.NewWithName(morm.SQLServer, "sqlserver://sa:c1UhH%5E%25h%25lWPqXS2%233tE@127.0.0.1:1433?database=master", "movies")
	AssertT(t, e == nil, e)

	t.Run("create_table", func(t *testing.T) {
		e := orm.CreateTable(mssql_user{}, "")
		AssertT(t, e == nil, e)
	})

	t.Run("insert_data", func(t *testing.T) {
		u := mssql_user{FirstName: "Richard"}
		e := orm.Insert(&u)
		AssertT(t, e == nil, e)
	})
}
