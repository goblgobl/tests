package factory

/*
A helper to build test factories
*/

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"src.goblgobl.com/utils/sqlite"
	"src.goblgobl.com/utils/typed"
)

type SQLStorage interface {
	JSON(input any) (any, error)
	MustExec(sql string, args ...any)
	RowToMap(sql string, args ...any) (typed.Typed, error)
	Placeholder(i int) string
}

// Factory where the [sqlite] connection is passed at runtime
type SQLiteProvider interface {
	WithDB(func(conn sqlite.Conn) error) error
}

var DB SQLStorage

type Table struct {
	Truncate func() Table
	Insert   func(args ...any) typed.Typed
}

type Sqlite struct {
	Truncate func(p SQLiteProvider) Sqlite
	Insert   func(p SQLiteProvider, args ...any) typed.Typed
}

func NewTable(name string, builder func(KV) KV, pks ...string) Table {
	keys, insertSQL, deleteSQL := buildSQL(name, builder, DB.Placeholder, pks...)

	t := Table{}
	t.Truncate = func() Table {
		DB.MustExec(deleteSQL)
		return t
	}

	t.Insert = func(args ...any) typed.Typed {
		obj := builder(ToKV(args))
		values := make([]any, len(obj))
		for i, k := range keys {
			value := obj[k]
			if jsonValue, ok := value.(JSON); ok && value != nil {
				var err error
				value, err = DB.JSON(jsonValue)
				if err != nil {
					panic(err)
				}
			}
			values[i] = value
		}
		row, err := DB.RowToMap(insertSQL, values...)
		if err != nil {
			panic(err)
		}
		return row
	}
	return t
}

func NewSqlite(name string, builder func(KV) KV, pks ...string) Sqlite {
	keys, insertSQL, deleteSQL := buildSQL(name, builder, sqlitePlaceholderFactory, pks...)

	t := Sqlite{}
	t.Truncate = func(p SQLiteProvider) Sqlite {
		p.WithDB(func(conn sqlite.Conn) error {
			conn.MustExec(deleteSQL)
			return nil
		})
		return t
	}

	t.Insert = func(p SQLiteProvider, args ...any) typed.Typed {
		obj := builder(ToKV(args))
		values := make([]any, len(obj))
		for i, k := range keys {
			value := obj[k]
			if jsonValue, ok := value.(JSON); ok && value != nil {
				var err error
				value, err = json.Marshal(jsonValue)
				if err != nil {
					panic(err)
				}
			}

			values[i] = value
		}

		var r typed.Typed
		p.WithDB(func(conn sqlite.Conn) error {
			var err error
			r, err = conn.RowToMap(insertSQL, values...)
			return err
		})
		return r
	}
	return t
}

func buildSQL(name string, builder func(KV) KV, placeholderFactory func(i int) string, pks ...string) ([]string, string, string) {
	obj := builder(KV{})
	keys := make([]string, len(obj))
	placeholders := make([]string, len(obj))

	i := 0
	for k := range obj {
		keys[i] = k
		placeholders[i] = placeholderFactory(i)
		i++
	}

	insertSQL := "insert into " + name + " (" + strings.Join(keys, ",") + ")"
	insertSQL += "\nvalues (" + strings.Join(placeholders, ",") + ")"
	if len(pks) > 0 {
		insertSQL += "\non conflict (" + strings.Join(pks, ",") + ") do update set "
		insertSQL += keys[0] + " = excluded." + keys[0]
		for _, k := range keys[1:] {
			insertSQL += ", " + k + " = excluded." + k
		}
	}
	insertSQL += " returning *"

	deleteSQL := "delete from " + name

	return keys, insertSQL, deleteSQL
}

type JSON map[string]any
type KV map[string]any

func ToKV(opts []any) KV {
	args := make(KV, len(opts)/2)
	for i := 0; i < len(opts); i += 2 {
		args[opts[i].(string)] = opts[i+1]
	}
	return args
}

func (kv KV) UUID(key string, dflt ...string) any {
	if value, exists := kv[key]; exists {
		return value.(string)
	}
	if len(dflt) == 1 {
		return dflt[0]
	}
	return nil
}

func (kv KV) Int(key string, dflt ...int) any {
	if value, exists := kv[key]; exists {
		return value.(int)
	}
	if len(dflt) == 1 {
		return dflt[0]
	}
	return nil
}

func (kv KV) Float(key string, dflt ...float64) any {
	if value, exists := kv[key]; exists {
		switch t := value.(type) {
		case int:
			return float64(t)
		case float32:
			return float64(t)
		case float64:
			return t
		default:
			panic(fmt.Sprintf("Invalid float64: %t", value))
		}
	}

	if len(dflt) == 1 {
		return dflt[0]
	}
	return nil
}

func (kv KV) UInt16(key string, dflt ...uint16) any {
	if value, exists := kv[key]; exists {
		switch v := value.(type) {
		case int:
			return uint16(v)
		case uint16:
			return v
		default:
			return uint16(reflect.ValueOf(value).Uint())
		}
	}
	if len(dflt) == 1 {
		return dflt[0]
	}
	return nil
}

func (kv KV) Bool(key string, dflt ...bool) any {
	if value, exists := kv[key]; exists {
		return value.(bool)
	}
	if len(dflt) == 1 {
		return dflt[0]
	}
	return nil
}

func (kv KV) String(key string, dflt ...string) any {
	if value, exists := kv[key]; exists {
		return value.(string)
	}
	if len(dflt) == 1 {
		return dflt[0]
	}
	return nil
}

func (kv KV) Strings(key string, dflt ...string) any {
	if value, exists := kv[key]; exists {
		switch value.(type) {
		case []string:
			return value
		case string:
			return []string{value.(string)}
		default:
			panic("invalid string/[]string value")
		}
	}
	if len(dflt) == 1 {
		return dflt[0]
	}
	return nil
}

func (kv KV) Time(key string, dflt ...time.Time) any {
	if value, exists := kv[key]; exists {
		return value.(time.Time)
	}
	if len(dflt) == 1 {
		return dflt[0]
	}
	return nil
}

func (kv KV) JSON(key string, dflt map[string]any) any {
	value, exists := kv[key]
	if !exists {
		value = dflt
	}
	if value == nil {
		return nil
	}

	switch value.(type) {
	case string:
		return value
	case []byte:
		return value
	case nil:
		return nil
	default:
		data, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		return data
	}
}

func sqlitePlaceholderFactory(i int) string {
	return "?" + strconv.Itoa(i+1)
}
