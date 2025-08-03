# Minimal Object Relational Model (MORM)

MORM (Minimal Object Relational Model) is a light weight object relational model. The idea is to handle simple `CRUD` operations as well as `creating` and `droppping` tables. Any query that is more complex than that I am an advocate of writing it by hand or use [SQLC](https://sqlc.dev/) an awesome productivity tool with great support


## Quick Start

```go

import "github.com/chapgx/morm"

type User struct {
  Id int `morm:"id integer primary"` // this tag notation will be executed exactly as it is
  FirstName string `morm:"first_name text"` // this gets execute exactly as it is using first_name as the field name
  LastName string // untagged fields will be added by default . In this case the field name is lastname
  Foo string `morm:":ignore"` // this is a directive tells the ORM to ignore this field
  Email Email // default behavior is to create a new table based on the struct
  Phone Phone `morm:":flatten"` // this will bring all the Phone struct fields to the User table
}


// this structure uses the struct field names as the table fields and the data types as well
type Email struct {
  Id int
  Address string
}


type Phone struct {
  Primary bool
  Number string
}


func main() {
  // takes engine and connection string, returns a *MORM and an error
  _, e := morm.New(SQLITE, "db.db")
  if e != nil {
    panic(e)
  }

  var u User
  // second parameter is table name if none passed the struct name is use, in this case users would be use
  e := morm.CreateTable(u, "")
  if e != nil {
    panic(e)
  }
}

```

The query for the example above would be

```sql

create table if not exists users (
  id integer primary,
  first_name text,
  lastname text,
  phone_primary integer check(phone_primary in(0,1)),
  phone_number text,
)

create table if not exists emails (
  id integer,
  address text,
)
```

Form the struct to the query to the engine executing the query that is the basic process of `MORM`


## DIRECTIVES

Special tags that declare behavior for a structure field beyond SQL context


| Directive | Usage | Description |
| --------- | -------- | ----------- |
| ignore | :ignore |  ignores the field   |
| flatten | :flatten | fields in nested structure are merge into the parent table (the table made out of the parent structure) |


## ENGINES

SQL engines supported and states of development


| Name | Supported | Plan To Support | State |
| -------| ---------| ------------| --------|
| SQLITE | YES | YES | Developement |
| SQLServer | NO | YES | Pending |
| Postgress | NO | YES | Pending |
| MySql | NO | YES | Pending |


## How To Use

By Default you don't have to add `MORM` tags for the structure to be transpile into it's SQL representation. In the case of no tags it will simple use the field name (this will be turn to lowercase) and it will try to represent the specify data type in it's SQL form. If the structure contains nested structures then adjacent tables will be created for them.


At the beginning of your process of before you start calling library functions you most specify the engine to be used. Sample below.

```go
import "github.com/chapgx/morm"

// engine and connection string passed in
_, e := morm.New(SQLITE, "db.db")


```


## Tags

The `MORM` tag is simple statement in SQL of the field. For example an ID of type integer would be tag as follow `morm:"id int primary"`, this would make the id an integer and the column primary.


Directives are special tags that expand beyond the context of SQL check the [Directives](#directives) sessions for more information.

## Samples

Common use case examples of `MORM`. All examples assume you have specify the engine/driver and connection string. For how to do that check [How To Use](#how-to-use)

`Create a table`

```go 
// engine declare as SQLITE

import "github.com/chapgx/morm"

type Plane struct {
  Number int `morm:"number text"`
  Weight float
  Active bool
  NeedService bool `morm:"need_service int check(need_service in(0,1))"`
}

// creates sql table out of Plane struct, by default the name is planes but you can overwrite with
// the second parameter
err := morm.CreateTable(Plane{}, "")

```

`Insert Record`

```go 
// engine declare as SQLITE

import "github.com/chapgx/morm"


plane := Plane{"BOING_00", 15.20}
// inters record into the database
err := morm.Insert(&plane)

plane1 := Plane{"BOING_01", 20}
// insert record into the database with a explicit table name
err := morm.InsertByName(&plane1, "planes_backup")

```

`Update Record`

```go 

import "github.com/chapgx/morm"
import "fmt"

plane := Plane{NeedService: true}

// create your filter which is an in code representation of a where clause
filter := morm.NewFilter()
filter.
  And("name", EQUAL, "BOING_07").
  Or("weight", GREATER, 20)

g := filter.Group()
g.
  And("active", EQUAL,  true).
  Or("need_service", EQUAL,  false)


result := Update(&plane, &filter, "NeedService")
if result.Error != nil {
  panic(result.Error)
}

fmt.Println("affected", result.RowsAffected)

```

The SQL would look like this

```sql
update planes
set need_service = 1
where name = 'BOING_07' or weight > 20
and (active = 1 or need_service = 0)
```


P.S
If filters equals nil it will make the update to all records in the database

`Delete Record`
For `Delete` filter is required and it will panic if non passed. This is to avoid deleting the entire table. Use `morm.Exec(query)` for something like that.

```go 

import "github.com/chapgx/morm"

filter := morm.NewFilter()
filter.And("weight", GREATER, 20)


rslt := DeleteByName("planes", &filters)
// handle error and check rows affected if needed

rslt = Delete(Plane{}, &filters)

```



## ROADMAP

- [x] CRUD Operations On simple structures.
- [ ] Full SQLITE support
- [ ] Full SQL Server support
