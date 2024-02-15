package tests

import (
	"os"
	"regexp"
	"strings"

	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/typed"
)

var PlaceholderPattern = regexp.MustCompile(`\$(\d+)`)

type TestableDB interface {
	Placeholder(i int) string
	IsNotFound(err error) bool
	RowToMap(sql string, args ...any) (typed.Typed, error)
	RowsToMap(sql string, args ...any) ([]typed.Typed, error)
}

func Row(db TestableDB, sql string, args ...any) typed.Typed {
	// no one's going to like this, but not sure how else to deal with it
	if db.Placeholder(0) == "?1" {
		sql = PlaceholderPattern.ReplaceAllString(sql, "?$1")
	}
	row, err := db.RowToMap(sql, args...)
	if err != nil {
		if db.IsNotFound(err) {
			return nil
		}
		panic(err)
	}
	return row
}

func Rows(db TestableDB, sql string, args ...any) []typed.Typed {
	// no one's going to like this, but not sure how else to deal with it
	if db.Placeholder(0) == "?1" {
		sql = PlaceholderPattern.ReplaceAllString(sql, "?$1")
	}
	rows, err := db.RowsToMap(sql, args...)
	if err != nil {
		panic(err)
	}
	return rows
}

func PG(dbName string) string {
	pg := os.Getenv("GOBL_TEST_PG")
	if pg == "" {
		pg = "postgres://" + dbName + "@localhost:5432"
	}
	return pg + "/" + dbName
}

func CR(dbName string) string {
	cr := os.Getenv("GOBL_TEST_CR")
	if cr == "" {
		cr = "postgres://root@localhost:26257"
	}
	return cr + "/" + dbName
}

func StorageType() string {
	env := strings.ToLower(os.Getenv("GOBL_TEST_STORAGE"))
	switch env {
	case "postgres":
		return "postgres"
	case "cockroach":
		return "cockroach"
	case "sqlite", "":
		return "sqlite"
	default:
		panic("Unknown GOBL_TEST_STORAGE value. Should be one of: pg, cr, sqlite (default)")
	}
}

func CaptureLog(fn func()) string {
	defer func() {
		log.Out = os.Stderr
	}()

	str := &strings.Builder{}
	log.Out = str
	fn()
	return str.String()
}
