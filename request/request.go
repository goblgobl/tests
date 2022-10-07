package request

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/utils/http"
	"src.goblgobl.com/utils/json"
	"src.goblgobl.com/utils/typed"

	"github.com/valyala/fasthttp"
)

func Req(t *testing.T) RequestBuilder {
	return RequestBuilder{
		t:       t,
		path:    "/",
		query:   make(url.Values),
		headers: make(map[string]string),
	}
}

// Most cases should generates a respins from a Req, but some cases will want
// to test an http.Response directly
func Response(t *testing.T, res http.Response) response {
	conn := &fasthttp.RequestCtx{}
	res.Write(conn)
	return Res(t, conn)
}

type RequestBuilder struct {
	t       *testing.T
	host    string
	body    string
	path    string
	method  string
	query   url.Values
	headers map[string]string
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
	return r.Header("Gobl-Project", id)
}

func (r RequestBuilder) Query(key string, value string) RequestBuilder {
	r.query.Add(key, value)
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

func (r RequestBuilder) Host(host string) RequestBuilder {
	r.host = host
	return r
}

func (r RequestBuilder) Get(handler interface{}) response {
	return r.Method("GET").Request(handler)
}

func (r RequestBuilder) Post(handler interface{}) response {
	return r.Method("POST").Request(handler)
}

func (r RequestBuilder) Put(handler interface{}) response {
	return r.Method("PUT").Request(handler)
}

func (r RequestBuilder) Delete(handler interface{}) response {
	return r.Method("DELETE").Request(handler)
}

func (r RequestBuilder) Request(handler interface{}) response {
	conn := r.Conn()

	switch h := handler.(type) {
	case func(*fasthttp.RequestCtx):
		h(conn)
		return Res(r.t, conn)
	default:
		panic(fmt.Sprintf("unknown handler type: %T", handler))
	}

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

	return &fasthttp.RequestCtx{
		Request: *request,
	}
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

	return response{
		t:             t,
		Json:          json,
		Headers:       headers,
		Body:          string(body),
		Status:        res.StatusCode(),
		ContentLength: res.Header.ContentLength(),
	}
}

type response struct {
	t             *testing.T
	Status        int
	Body          string
	Json          typed.Typed
	ContentLength int
	Headers       map[string]string
}

func (r response) ExpectCode(expected int) response {
	r.t.Helper()
	assert.Equal(r.t, r.Json.Int("code"), expected)
	return r
}

func (r response) ExpectNotFound(code ...int) response {
	r.t.Helper()
	assert.Equal(r.t, r.Status, 404)
	if len(code) == 1 {
		r.ExpectCode(code[0])
	}
	return r
}

func (r response) ExpectNotAuthorized(code ...int) response {
	r.t.Helper()
	assert.Equal(r.t, r.Status, 401)
	if len(code) == 1 {
		r.ExpectCode(code[0])
	}
	return r
}

func (r response) ExpectInvalid(code ...int) response {
	r.t.Helper()
	assert.Equal(r.t, r.Status, 400)
	if len(code) == 1 {
		r.ExpectCode(code[0])
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

func (r response) ExpectValidation(expected ...any) map[string][]typed.Typed {
	r.t.Helper()
	assert.Equal(r.t, r.Status, 400)
	r.ExpectCode(20000)

	o := r.Json.Objects("invalid")
	lookup := make(map[string][]typed.Typed, len(o))
	for _, o := range o {
		field := o.String("field")
		lookup[field] = append(lookup[field], o)
	}

	valid := true
	for i := 0; i < len(expected); i += 2 {
		field := expected[i].(string)
		expectedCode := expected[i+1].(int)
		actuals := lookup[field]
		for _, actual := range actuals {
			actualCode := actual.Int("code")
			if expectedCode == actualCode {
				break
			}
			r.t.Errorf("Expect validation code for field '%s' to be %d, got %d", field, expectedCode, actualCode)
			valid = false
		}
	}

	if !valid {
		r.t.FailNow()
	}

	return lookup
}

func (r response) OK() response {
	r.t.Helper()
	if r.Status != 200 && r.Status != 201 && r.Status != 204 {
		r.t.Errorf("Expect 200/201/204 status code, got: %d", r.Status)
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
