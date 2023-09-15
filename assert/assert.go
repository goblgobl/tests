// A wrapper around *testing.T. I hate the if a != b { t.ErrorF(....) } pattern.
package assert

import (
	"bytes"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

// a == b
func Equal[T comparable](t *testing.T, actual T, expected T) {
	t.Helper()
	if actual != expected {
		t.Errorf("\nexpected: '%v'\nto equal: '%v'", expected, actual)
		t.FailNow()
	}
}

// a != b
func NotEqual[T comparable](t *testing.T, actual T, expected T) {
	t.Helper()
	if actual == expected {
		t.Errorf("\nexpected: '%v'\nto not equal: '%v'", expected, actual)
		t.FailNow()
	}
}

func Bytes(t *testing.T, actual []byte, expected []byte) {
	t.Helper()
	if bytes.Compare(actual, expected) != 0 {
		t.Errorf("\nexpected: '%v'\nto equal: '%v'", expected, actual)
		t.FailNow()
	}
}

// Two lists are equal (same length & same values in the same order)
func List[T comparable](t *testing.T, actuals []T, expecteds []T) {
	t.Helper()
	Equal(t, len(actuals), len(expecteds))

	for i, actual := range actuals {
		Equal(t, actual, expecteds[i])
	}
}

// A value is nil
func Nil(t *testing.T, actual any) {
	t.Helper()
	if actual != nil {
		v := reflect.ValueOf(actual)
		kind := v.Kind()
		if (kind != reflect.Ptr && kind != reflect.Map) || !v.IsNil() {
			t.Errorf("expected %v to be nil", actual)
			t.FailNow()
		}
	}
}

// A value is not nil
func NotNil(t *testing.T, actual any) {
	t.Helper()
	if actual == nil {
		t.Errorf("expected %v to be not nil", actual)
		t.FailNow()
	}
}

// A value is true
func True(t *testing.T, actual bool) {
	t.Helper()
	if !actual {
		t.Error("expected true, got false")
		t.FailNow()
	}
}

// A value is false
func False(t *testing.T, actual bool) {
	t.Helper()
	if actual {
		t.Error("expected false, got true")
		t.FailNow()
	}
}

// The string contains the given value
func StringContains(t *testing.T, actual string, expected string) {
	t.Helper()
	if !strings.Contains(actual, expected) {
		t.Errorf("\nexpected: '%s'\nto contain: '%s'", actual, expected)
		t.FailNow()
	}
}

func Error(t *testing.T, actual error, expected error) {
	t.Helper()
	if !errors.Is(actual, expected) {
		t.Errorf("expected '%s' to be '%s'", actual, expected)
		t.FailNow()
	}
}

func Nowish(t *testing.T, actual time.Time) {
	t.Helper()
	diff := math.Abs(time.Now().UTC().Sub(actual).Seconds())
	if diff > 1 {
		t.Errorf("expected '%s' to be nowish", actual)
		t.FailNow()
	}
}

func Timeish(t *testing.T, actual time.Time, expected time.Time) {
	t.Helper()
	diff := math.Abs(expected.Sub(actual).Seconds())
	if diff > 1 {
		t.Errorf("expected '%s' to be around '%s'", actual, expected)
		t.FailNow()
	}
}

func Fail(t *testing.T, fmt string, args ...interface{}) {
	t.Helper()
	t.Errorf(fmt, args...)
	t.FailNow()
}

type Numeric interface {
	~float32 | ~float64 |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func Delta[T Numeric](t *testing.T, actual T, expected T, delta T) {
	t.Helper()
	if actual < expected-delta || actual > expected+delta {
		t.Errorf("\nexpected: '%v'\nto be within %v of equal: '%v'", actual, delta, expected)
		t.FailNow()
	}
}
