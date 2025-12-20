# pgsql-builder

This package generates Postgre SQL queries based on a struct instance. The concept is to define a struct, create a corresponding table to store its instances, and generate queries for managing the rows in that table, such as creating, updating, deleting, and selecting records.

> ⚠️ The project might be good for small prototyping and showcase of reflection package. It is not recommended to use it in production. Please use a more mature library instead.

The following queries can be generated:

* `CREATE TABLE`
* `DROP TABLE`
* `INSERT`
* `UPDATE ... WHERE id = ...`
* `INSERT ... ON CONFLICT UPDATE ...` (upsert)
* `SELECT ... WHERE id = ...`
* `DELETE ... WHERE id = ...`
* `SELECT ... WHERE ...`
* `DELETE ... WHERE ...`
* `UPDATE ... WHERE ...`


## How to use

### TL;DR

Check the code in `main_test.go` file that contains tests for all use cases.

### Defining a struct

Create a struct to define an object to be stored in a database table.  In the example below, let's create a `Product`.

As of now, a field called `ID` is required.  Also, the corresponding table column name that is generated for it is always prefixed with struct name. Hence, for `ID` field in `Product` that would be `product_id`.

There is another field called `Flags` that could be added, and it treated the same (meaning `product_flags` is generated, instead of `flags`).

````go
type Product struct {
  ID int64
  Flags int64
  Name string
  Description string `sql:"type:varchar(2000)"` // "type" is used to force a specific string type for a column
  Code string `sql:"uniq"` // "uniq" tells the module to make this column uniq
  ProductionYear int
  CreatedByUserID int64
  LastModifiedByUserID int64
}
````

#### Field tags

In the above definition, a special tag `sql` is used to add specific configuration for the column when creating a table.

| Tag key | Description |
|---|-----------|
| `uniq` | When passed, the column will get a `UNIQUE` constraint|
| `type` | Overwrites default `VARCHAR(255)` column type for string field. Possible values are: `TEXT`, `BPCHAR(X)`, `CHAR(X)`, `VARCHAR(X)`, `CHARACTER VARYING(X)`, `CHARACTER(X)` where `X` is the size. See [PostgreSQL character types](https://www.postgresql.org/docs/current/datatype-character.html) for more information. |

A different than `sql` tag can be used by passing `TagName` in `StructSQLOptions{}` when calling `NewStructSQL` function (see below.)

### Create a controller for the struct

To generate an SQL query based on a struct, a `StructSQL` object is used.  One per struct.

````go

import (
  sqlbuilder "github.com/keenbytes/pgsql-builder"
)

(...)

b := sqlbuilder.New(Product{}, sqlbuilder.Options{})
````

#### Options

There are certain options that can be provided to the `New` function which change the functionality of the object.  See the table below.

| Key                          | Type | Description                                                                                                                                                       |
|------------------------------|---|-------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| TableNamePrefix              | `string` | Prefix for the table name, eg. `myprefix_`                                                                                                                        |
| StructName                   | `string` | Table name is created out of the struct name, eg. for `MyProduct` that would be `my_product`. It is possible to overwrite the struct name, and further table name. |
| TagName                      | `string` | Uses a different tag than `sql`.  It is very useful when another module uses this module.                                                                         |

### Get SQL queries

Use any of the following functions to get a desired SQL query. 

| Function                   |
|----------------------------|
| `DropTable()`              |
| `CreateTable()`            |
| `Insert()`                 |
| `UpdateById()`             |
| `InsertOnConflictUpdate()` |
| `SelectById()`             |
| `DeleteById()`             |
| `Select(order []string, limit int, offset int, filters *Filters)`                 |
| `SelectCount(filters *Filters)`                 |
| `Delete(filters *Filters)`                 |
| `DeleteReturningID(filters *Filters)` |
| `Update(values map[string]interface{}, filters *Filters)`                 |

### Get SQL queries with conditions

It is possible to generate queries such as `SELECT`, `DELETE` or `UPDATE` with conditions based on fields.  In the following examples below, all the conditions (called "filters" in the code) are optional - there is no need to pass them.

#### SELECT

````go
// SELECT * FROM products WHERE (created_by_user_id=$1 AND name=$2) OR (product_year > $3
// AND product_year > $4 AND last_modified_by_user_id IN ($5,$6,$7,$8))
// ORDER BY production_year ASC, name ASC
// LIMIT 100 OFFSET 10
sql := b.Select(
  []string{"ProductionYear", "asc", "Name", "asc"},
  100, 10, 
  &b.Filters{
    "CreatedByUserID": {Op: OpEqual, Val: 4},
    "Name":            {Op: OpEqual, Val: "Magic Sock"},
    Raw: {
      Op: OpOR, 
      Val: []interface{}{
        ".ProductYear > ? AND .ProductYear < ? AND .LastModifiedByUserID(?)",
        // The below values are not important, but the overall number of args must match question marks.
        0,
        0,
        []int{0,0,0,0}, // this list must contain the same number of items as values
      },
    },
  })
````

#### SELECT COUNT(*)

````
// Use SelectCount without th first 3 arguments to get SELECT COUNT(*)
````

#### DELETE

````go
// DELETE FROM products WHERE (created_by_user_id=$1 AND name=$2) OR (product_year > $3
// AND product_year > $4 AND last_modified_by_user_id IN ($5,$6,$7,$8))
sql := b.Delete(
  &b.Filters{
    "CreatedByUserID": {Op: OpEqual, Val: 4},
    "Name": {Op: OpEqual, Val: "Magic Sock"},
    Raw: {
      Op: OpOR,
      Val: []interface{}{
        ".ProductYear > ? AND .ProductYear < ? AND .LastModifiedByUserID(?)",
        // The below values are not important, but the overall number of args must match question marks.
        0,
        0,
        []int
      }{0,0,0,0}, // this list must contain the same number of items as values
    },
  })
````

#### UPDATE

````go
// UPDATE products SET production_year=$1, last_modified_by_user_id=$2
// WHERE name LIKE $3;
sql := b.Update(
  map[string]interface{
    "ProductionYear": 1984,
    "LastModifiedByUserID": 13
  },
  &b.Filters{
    Raw: {
      Op: OpAND,
      Val: []interface{} {
        ".Name LIKE ?",
		0,
      },
    },
  })
````
