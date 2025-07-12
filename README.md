# Minimal Object Relational Model (MORM)

MORM (Minimal Object Relational Model) is a light weight object relational model. The idea is to handle simple `CRUD` operations as well as `creating` and `droppping` tables. Any query that is more complex than that I am an advocate of writing it by hand or use [SQLC](https://sqlc.dev/) an awesome productivity tool with great support


## Quick Example

```go

type User struct {
  Id int `morm:"id integer primary"` // this tag notation will be executed exactly as it is
  FirstName string `morm:"first_name text"` // this gets execute exactly as it is using first_name as the field name
  LastName string // untagged fields will be added by default . In this case the field name is lastname
  Foo string `morm:":ignore"` // this is a directive tells the ORM to ignore this field
  Email Email // default behavior is to create a new table based on the struct
}


// this structure uses the struct field names as the table fields and the data types as well
type Email struct {
  Id int
  Address string
}


func main() {
  // takes engine and connection string, returns a *MORM and an error
  _, e := New(SQLITE, "db.db")
  if e != nil {
    panic(e)
  }

  var u User
  // second parameter is table name if none passed the struct name is use, in this case users would be use
  e := CreateTable(u, "")
  if e != nil {
    panic(e)
  }
}

```

The sample query for the example above would be

```sql

create table if not exists users (
  id integer primary,
  first_name text,
  lastname text,
)

create table if not exists emails (
  id integer,
  address text,
)
```


## DIRECTIVES

| Directive | Usage | Description |
| --------- | -------- | ----------- |
| ignore | :ignore |  ignores the field   |
| flatten | :flatten | fields in nested structure are merge into the parent table |



## ROADMAP
