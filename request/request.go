package request

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/utils/http"
	"src.goblgobl.com/utils/json"
	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/typed"

	"github.com/valyala/fasthttp"
)

type Handler func(*fasthttp.RequestCtx)

func Req(t *testing.T) RequestBuilder {
	return RequestBuilder{
		t:          t,
		path:       "/",
		query:      make(url.Values),
		headers:    make(map[string]string),
		userValues: make(map[string]any),
	}
}

// Most cases should generates a response from a Req, but some cases will want
// to test an http.Response directly
func Response(t *testing.T, res http.Response) response {
	conn := &fasthttp.RequestCtx{}
	res.Write(conn, log.Noop{})
	return Res(t, conn)
}

type RequestBuilder struct {
	t          *testing.T
	host       string
	body       string
	path       string
	method     string
	query      url.Values
	headers    map[string]string
	userValues map[string]any
}

func (r RequestBuilder) Path(path string) RequestBuilder {
	r.path = path
	return r
}

func (r RequestBuilder) Method(method string) RequestBuilder {
	r.method = method
	return r
}

func (r RequestBuilder) Header(key string, value string) RequestBuilder {
	r.headers[key] = value
	return r
}

func (r RequestBuilder) ProjectId(id string) RequestBuilder {
	return r.Header("Project", id)
}

func (r RequestBuilder) QueryMap(query map[string]string) RequestBuilder {
	for k, v := range query {
		r.query.Add(k, v)
	}
	return r
}

func (r RequestBuilder) Query(query ...string) RequestBuilder {
	for i := 0; i < len(query); i += 2 {
		r.query.Add(query[i], query[i+1])
	}

	return r
}

func (r RequestBuilder) Body(body any) RequestBuilder {
	if s, ok := body.(string); ok {
		r.body = s
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		r.body = string(data)
	}
	return r
}

func (r RequestBuilder) UserValue(key string, value any) RequestBuilder {
	r.userValues[key] = value
	return r
}

func (r RequestBuilder) Host(host string) RequestBuilder {
	r.host = host
	return r
}

func (r RequestBuilder) Get(handler Handler) response {
	return r.Method("GET").Request(handler)
}

func (r RequestBuilder) Post(handler Handler) response {
	return r.Method("POST").Request(handler)
}

func (r RequestBuilder) Put(handler Handler) response {
	return r.Method("PUT").Request(handler)
}

func (r RequestBuilder) Delete(handler Handler) response {
	return r.Method("DELETE").Request(handler)
}

func (r RequestBuilder) Request(handler Handler) response {
	conn := r.Conn()
	handler(conn)
	return Res(r.t, conn)
}

func (r RequestBuilder) Conn() *fasthttp.RequestCtx {
	request := new(fasthttp.Request)
	if body := r.body; body != "" {
		request.AppendBodyString(body)
	}
	header := new(fasthttp.RequestHeader)
	header.SetMethod(r.method)
	for key, value := range r.headers {
		header.Set(key, value)
	}
	request.Header = *header

	uri := "http://"
	if h := r.host; h != "" {
		uri += h
	} else {
		uri += "test.goblgobl.local"
	}
	uri += r.path
	if len(r.query) > 0 {
		uri += "?" + r.query.Encode()
	}
	request.SetRequestURI(uri)

	ctx := &fasthttp.RequestCtx{
		Request: *request,
	}

	for key, value := range r.userValues {
		ctx.SetUserValue(key, value)
	}

	return ctx
}

func Res(t *testing.T, conn *fasthttp.RequestCtx) response {
	res := conn.Response

	body := res.Body()
	// might not be json, just ignore if so, let the test deal with it
	json, _ := typed.Json(body)

	headers := make(map[string]string)
	res.Header.VisitAll(func(key []byte, value []byte) {
		headers[string(key)] = string(value)
	})

	status := res.StatusCode()

	// if we have a validation error, let's parse them into a lookup
	// of field => code
	var validations map[string][]typed.Typed
	if status == 400 && json.Int("code") == 2004 {
		o := json.Objects("invalid")
		validations = make(map[string][]typed.Typed, len(o))
		for _, o := range o {
			field := o.String("field")
			validations[field] = append(validations[field], o)
		}
	}

	return response{
		t:             t,
		Json:          json,
		Headers:       headers,
		Bytes:         body,
		Body:          string(body),
		Status:        status,
		Validations:   validations,
		ContentLength: res.Header.ContentLength(),
	}
}

type response struct {
	t             *testing.T
	Err           error
	Status        int
	Bytes         []byte
	Body          string
	Json          typed.Typed
	ContentLength int
	Headers       map[string]string
	Validations   map[string][]typed.Typed
}

func (r response) SHA256() string {
	sum := sha256.Sum256(r.Bytes)
	return fmt.Sprintf("%x", sum)
}

func (r response) ExpectCode(expected int) response {
	r.t.Helper()
	actual := r.Json.Int("code")
	if actual != expected {
		assert.Fail(r.t, "expected code: %d, got %d\r%s", expected, actual, r.Body)
	}
	return r
}

func (r response) ExpectNotFound(code ...int) response {
	r.t.Helper()
	r.ExpectStatus(404)
	if len(code) == 1 {
		r.ExpectCode(code[0])
	}
	return r
}

func (r response) ExpectNotAuthorized(code ...int) response {
	r.t.Helper()
	r.ExpectStatus(401)
	if len(code) == 1 {
		r.ExpectCode(code[0])
	}
	return r
}

func (r response) ExpectInvalid(code ...int) response {
	r.t.Helper()
	r.ExpectStatus(400)
	if len(code) == 1 {
		r.ExpectCode(code[0])
	}
	return r
}

func (r response) ExpectStatus(expected int) response {
	r.t.Helper()
	if r.Status != expected {
		assert.Fail(r.t, "\nexpected status code: %d\n\t\t got: %d\nbody: %s", expected, r.Status, r.Body)
	}
	return r
}

func (r response) Inspect() response {
	fmt.Printf("status: %d\n", r.Status)
	for k, v := range r.Headers {
		fmt.Printf("%s = %s\n", k, v)
	}
	fmt.Println(r.Body)
	return r
}

func (r response) ExpectValidation(expected ...any) response {
	r.t.Helper()
	r.ExpectStatus(400)
	r.ExpectCode(2004)

	valid := true
	lookup := r.Validations
	for i := 0; i < len(expected); i += 2 {
		found := false
		field := expected[i].(string)
		expectedCode := expected[i+1].(int)
		actuals := lookup[field]
		for _, actual := range actuals {
			found = true
			actualCode := actual.Int("code")
			if expectedCode == actualCode {
				break
			}
			r.t.Errorf("Expect validation code for field '%s' to be %d, got %d", field, expectedCode, actualCode)
			valid = false
		}

		if !found {
			r.t.Errorf("No validation error for field '%s'", field)
			valid = false
		}
	}

	if !valid {
		r.t.FailNow()
	}

	return r
}

func (r response) ExpectNoValidation(fields ...string) response {
	r.t.Helper()
	valid := true
	for _, field := range fields {
		if invalid, exists := r.Validations[field]; exists {
			valid = false
			r.t.Errorf("Did not expect an error for field: '%s', got: '%v'", field, invalid)
		}
	}
	if !valid {
		r.t.FailNow()
	}
	return r
}

func (r response) OK() response {
	r.t.Helper()
	if r.Status != 200 && r.Status != 201 && r.Status != 204 {
		r.t.Errorf("Expect 200/201/204 status code, got: %d\n%s\n%v", r.Status, r.Body, r.Err)
		r.t.FailNow()
	}
	return r
}

func (r response) JSON() typed.Typed {
	r.t.Helper()
	return typed.Must([]byte(r.Body))
}

func (r response) Header(name string, expected string) response {
	r.t.Helper()
	name = strings.Title(name)
	assert.Equal(r.t, r.Headers[name], expected)
	return r
}
