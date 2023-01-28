package assert

/*
Helper to validation a src.goblgobl.com/utils/validation.Result
but using reflection so we don't create a cyclical dependency
(ya, that's normal...)
*/

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

type v struct {
	t      *testing.T
	json   []byte
	errors []map[string]any
}

func Validation(t *testing.T, result any) *v {
	e1 := reflect.ValueOf(result).MethodByName("Errors").Call(nil)[0]
	data, err := json.MarshalIndent(e1.Interface(), "", " ")
	if err != nil {
		panic(err)
	}

	var e2 []map[string]any
	if err := json.Unmarshal(data, &e2); err != nil {
		panic(err)
	}

	return &v{
		t:      t,
		json:   data,
		errors: e2,
	}
}

func (v *v) Fieldless(meta any) *v {
	return v.Field("", meta)
}

// Can pass:
// a code, e.g. Field("name", 10)
// a code with data, e.g. Field("name", 10, map[string]any{max: 20})
// anything else that we'll json serialize and compare with the error, e.g: Field(name, validation.Invalid{Code: 300})
func (v *v) Field(expectedField string, invalid ...any) *v {
	t := v.t
	t.Helper()

	var expectedCode int
	var expectedData map[string]any
	{
		if code, ok := invalid[0].(int); ok {
			expectedCode = code
			if len(invalid) == 2 {
				expectedData = invalid[1].(map[string]any)
			}
		} else {
			bytes, err := json.Marshal(invalid[0])
			if err != nil {
				panic(err)
			}
			var expectedError map[string]any
			if err := json.Unmarshal(bytes, &expectedError); err != nil {
				panic(err)
			}
			expectedCode = int(expectedError["code"].(float64))
			expectedData, _ = expectedError["data"].(map[string]any)
		}
	}

	for _, error := range v.errors {
		data := error["data"]
		field := error["field"]

		if field == nil {
			field = ""
		}

		if field != expectedField {
			continue
		}

		if int(error["code"].(float64)) != expectedCode {
			continue
		}

		if (data == nil && expectedData != nil) || (data != nil && expectedData == nil) {
			continue
		}

		if data != nil && expectedData != nil && !reflect.DeepEqual(data.(map[string]any), expectedData) {
			continue
		}

		return v
	}

	err := "\nexpected validation error:\n"
	if expectedField != "" {
		err += fmt.Sprintf("  field=%s\n", expectedField)
	}
	err += fmt.Sprintf("  code=%d\n", expectedCode)
	err += fmt.Sprintf("  data=%v\n\n", expectedData)
	err += fmt.Sprintf("got: %s", string(v.json))
	t.Error(err)
	t.FailNow()
	return v
}

func (v *v) FieldMessage(expectedField string, expectedMessage string) *v {
	t := v.t
	t.Helper()

	for _, error := range v.errors {
		field := error["field"]

		if field != expectedField {
			continue
		}
		if error["error"].(string) != expectedMessage {
			continue
		}
		return v
	}

	err := "\nexpected validation error message:\n"
	err += fmt.Sprintf("  field=%s\n", expectedField)
	err += fmt.Sprintf("  message=%s\n", expectedMessage)
	err += fmt.Sprintf("got: %s", string(v.json))
	t.Error(err)
	t.FailNow()
	return v
}

func (v *v) FieldsHaveNoErrors(noFields ...string) *v {
	t := v.t
	t.Helper()

	for _, error := range v.errors {
		field := error["field"]
		if field == nil {
			continue
		}
		for _, noField := range noFields {
			if field == noField {
				t.Errorf("Expected no error for field '%s', but got:\n%v", field, error)
				t.FailNow()
			}
		}
	}
	return v
}
