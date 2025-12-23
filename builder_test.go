package pgsqlbuilder

import (
	"testing"
)

type TestStruct struct {
	ID             int64  `json:"id"`
	Flags          int64  `json:"flags"`
	PrimaryEmail   string `json:"email"`
	EmailSecondary string `json:"email2"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Age            int    `json:"age"`
	Price          int    `json:"price"`
	PostCode       string `json:"post_code"`
	PostCode2      string `json:"post_code2"`
	Password       string `json:"password"`
	CreatedBy      int64  `json:"created_by"`
	Key            string `json:"key" sql:"uniq type:varchar(2000)"`
}

var testStructObj = &TestStruct{}

func TestSQLQueries(t *testing.T) {
	h := New(testStructObj, Options{})

	got := h.DropTable()
	want := `DROP TABLE IF EXISTS "test_struct";`
	if got != want {
		t.Fatalf("Want %v\ngot %v", want, got)
	}

	got = h.CreateTable()
	want = `CREATE TABLE IF NOT EXISTS "test_struct" ("id" SERIAL PRIMARY KEY,"flags" BIGINT NOT NULL DEFAULT 0,` +
		`"primary_email" VARCHAR(255) NOT NULL DEFAULT '',"email_secondary" VARCHAR(255) NOT NULL DEFAULT '',` +
		`"first_name" VARCHAR(255) NOT NULL DEFAULT '',"last_name" VARCHAR(255) NOT NULL DEFAULT '',` +
		`"age" BIGINT NOT NULL DEFAULT 0,"price" BIGINT NOT NULL DEFAULT 0,"post_code" VARCHAR(255) NOT NULL DEFAULT '',` +
		`"post_code2" VARCHAR(255) NOT NULL DEFAULT '',"password" VARCHAR(255) NOT NULL DEFAULT '',` +
		`"created_by" BIGINT NOT NULL DEFAULT 0,"key" VARCHAR(2000) NOT NULL DEFAULT '' UNIQUE);`
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}
}

func TestSQLInsertQueries(t *testing.T) {
	h := New(testStructObj, Options{})

	got := h.Insert()
	want := `INSERT INTO "test_struct"("flags","primary_email","email_secondary","first_name","last_name",` +
		`"age","price","post_code","post_code2","password","created_by","key") ` +
		`VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING "id";`
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}
}

func TestSQLUpdateByIDQueries(t *testing.T) {
	h := New(testStructObj, Options{})

	got := h.UpdateByID()
	want := `UPDATE "test_struct" SET "flags"=$1,"primary_email"=$2,"email_secondary"=$3,"first_name"=$4,"last_name"=$5,` +
		`"age"=$6,"price"=$7,"post_code"=$8,"post_code2"=$9,"password"=$10,"created_by"=$11,"key"=$12 WHERE "id" = $13;`
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}
}

func TestSQLInsertOnConflictUpdateQueries(t *testing.T) {
	h := New(testStructObj, Options{})

	got := h.InsertOnConflictUpdate()
	want := `INSERT INTO "test_struct"("id","flags","primary_email","email_secondary","first_name","last_name",` +
		`"age","price","post_code","post_code2","password","created_by","key") ` +
		`VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ` +
		`ON CONFLICT ("id") DO UPDATE SET ` +
		`"flags"=$14,"primary_email"=$15,"email_secondary"=$16,"first_name"=$17,"last_name"=$18,"age"=$19,` +
		`"price"=$20,"post_code"=$21,"post_code2"=$22,"password"=$23,"created_by"=$24,"key"=$25 ` +
		`RETURNING "id";`
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}
}

func TestSQLDeleteQueries(t *testing.T) {
	h := New(testStructObj, Options{})

	got := h.DeleteByID()
	want := `DELETE FROM "test_struct" WHERE "id" = $1;`
	if got != want {
		t.Fatalf("\nwant %v\ngot %v", want, got)
	}
}

func TestSQLSelectQueries(t *testing.T) {
	h := New(testStructObj, Options{})

	selectPrefix := `SELECT "id","flags","primary_email","email_secondary","first_name","last_name",` +
		`"age","price","post_code","post_code2","password","created_by","key" FROM "test_struct"`

	got := h.SelectByID()
	want := selectPrefix + ` WHERE "id" = $1;`
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}

	got, _ = h.Select(nil, 67, 13, nil)
	want = selectPrefix + " LIMIT 67 OFFSET 13;"
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}

	got, _ = h.Select([]string{"EmailSecondary", "desc", "Age", "asc"}, 67, 13, &Filters{
		"Price":     {Op: OpEqual, Val: 4444},
		"PostCode2": {Op: OpEqual, Val: "11-111"},
	})
	want = selectPrefix + ` WHERE "post_code2"=$1 AND "price"=$2 ORDER BY "email_secondary" DESC,"age" ASC LIMIT 67 OFFSET 13;`
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}

	got, _ = h.Select([]string{"EmailSecondary", "asc", "Age", "desc"}, 1, 3, &Filters{
		"Price":     {Op: OpEqual, Val: 33},
		"PostCode2": {Op: OpEqual, Val: "11-111"},
		Raw: {
			Op: OpOR,
			Val: []interface{}{
				".Price=? OR (.EmailSecondary=? AND .Age IN (?)) OR (.Age IN (?)) OR (.EmailSecondary IN (?))",
				// We do not really care about the values, the query contains $x only symbols
				// However, we need to pass either value or an array so that an array can be extracted into multiple $x's
				0,
				0,
				[]int{0, 0, 0, 0},
				[]int{0, 0, 0},
				[]int{0, 0},
			},
		},
	})
	want = selectPrefix + ` WHERE` +
		` ("post_code2"=$1 AND "price"=$2) OR ("price"=$3 OR ("email_secondary"=$4 AND "age" IN ($5,$6,$7,$8)) OR ("age" IN ($9,$10,$11)) OR ("email_secondary" IN ($12,$13)))` +
		` ORDER BY "email_secondary" ASC,"age" DESC LIMIT 1 OFFSET 3;`
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}

	got, _ = h.Select([]string{"EmailSecondary", "desc"}, 61, 19, &Filters{
		"Price":     {Op: OpGreater, Val: 4443},
		"PostCode2": {Op: OpLike, Val: "11%"},
		"Age":       {Op: OpLowerOrEqual, Val: 99},
		"FirstName": {Op: OpMatch, Val: "^[A-Z][a-z]+$"},
		"CreatedBy": {Op: OpGreaterOrEqual, Val: 100},
		"Flags":     {Op: OpBit, Val: 8},
	})
	want = selectPrefix + ` WHERE "age"<=$1 AND "created_by">=$2 AND "first_name" ~ $3 AND "flags"&$4>0 AND "post_code2" LIKE $5 AND "price">$6` +
		` ORDER BY "email_secondary" DESC LIMIT 61 OFFSET 19;`
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}
}

func TestSQLSelectCountQueries(t *testing.T) {
	h := New(testStructObj, Options{})

	got, _ := h.SelectCount(&Filters{
		"Price": {Op: OpEqual, Val: 4444},
	})
	want := `SELECT COUNT(*) AS cnt FROM "test_struct" WHERE "price"=$1;`
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}
}

func TestSQLDeleteWithFiltersQueries(t *testing.T) {
	h := New(testStructObj, Options{})

	got, _ := h.Delete(&Filters{
		"Price": {Op: OpEqual, Val: 4444},
	})
	want := `DELETE FROM "test_struct" WHERE "price"=$1;`
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}

	got, _ = h.Delete(
		&Filters{
			"Price": {Op: OpEqual, Val: 4444},
			Raw: {
				Op: OpAND,
				Val: []interface{}{
					".Price=? OR .EmailSecondary=? OR .Age IN (?)",
					0,
					0,
					[]int{0, 0, 0},
				},
			},
		})
	want = `DELETE FROM "test_struct" WHERE ("price"=$1) AND ` +
		`("price"=$2 OR "email_secondary"=$3 OR "age" IN ($4,$5,$6));`
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}

	got, _ = h.Delete(
		&Filters{
			Raw: {
				Val: []interface{}{
					".Price=? OR .EmailSecondary=? OR .Age IN (?)",
					0,
					0,
					[]int{0, 0, 0},
				},
			},
		})
	want = `DELETE FROM "test_struct" WHERE ("price"=$1 OR "email_secondary"=$2 OR "age" IN ($3,$4,$5));`
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}

	got, _ = h.DeleteReturningID(
		&Filters{
			Raw: {
				Val: []interface{}{
					".Price=? OR .EmailSecondary=? OR .Age IN (?)",
					0,
					0,
					[]int{0, 0, 0},
				},
			},
		})
	want = `DELETE FROM "test_struct" WHERE ("price"=$1 OR "email_secondary"=$2 OR "age" IN ($3,$4,$5)) RETURNING "id";`
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}
}

func TestSQLUpdateQueries(t *testing.T) {
	h := New(testStructObj, Options{})

	got, _ := h.Update(
		map[string]interface{}{"Price": 1234, "PostCode2": "12-345"},
		&Filters{
			"PrimaryEmail": {Op: OpEqual, Val: "primary@example.com"},
		},
	)
	want := `UPDATE "test_struct" SET "post_code2"=$1,"price"=$2 WHERE "primary_email"=$3;`
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}

	got, _ = h.Update(
		map[string]interface{}{"FirstName": "Jane", "LastName": "Doe"},
		&Filters{
			"PostCode": {Op: OpEqual, Val: "11-111"},
		},
	)
	want = `UPDATE "test_struct" SET "first_name"=$1,"last_name"=$2 WHERE "post_code"=$3;`
	if got != want {
		t.Fatalf("\nwant %v\ngot  %v", want, got)
	}
}
