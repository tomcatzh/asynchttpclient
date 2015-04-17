package asynchttpclient

import (
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

var server = "127.0.0.1:31712"

func one(w http.ResponseWriter, r *http.Request) {
	time.Sleep(300 * time.Millisecond)
	io.WriteString(w, "1")
}

func two(w http.ResponseWriter, r *http.Request) {
	time.Sleep(150 * time.Millisecond)
	io.WriteString(w, "2")
}

func three(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "3")
}

func startServer() {
	http.HandleFunc("/1", one)
	http.HandleFunc("/2", two)
	http.HandleFunc("/3", three)

	go http.ListenAndServe(server, nil)
}

func getBody(resp *http.Response) (string, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

type result struct {
	Err  error
	Resp *http.Response
	Body string
}

type testRespHandler struct {
	ResultCh chan result
}

func newTestRespHandler(resultSize int) *testRespHandler {
	r := &testRespHandler{}
	r.ResultCh = make(chan result, resultSize)

	return r
}

func (h *testRespHandler) handle(err error, resp *http.Response) {
	var r result

	if err != nil {
		r.Err = err
	} else if resp != nil {
		r.Resp = resp
		r.Body, r.Err = getBody(resp)

	}

	h.ResultCh <- r
}

func TestGetSingleConcurrency(t *testing.T) {
	h := newTestRespHandler(3)
	c := AsyncHttpClient{Concurrency: 1}

	Convey("Test Async Get with Concurrency limit 1", t, func() {
		c.Get("http://"+server+"/1", h.handle)
		c.Get("http://"+server+"/2", h.handle)
		c.Get("http://"+server+"/3", h.handle)

		// There's only one Concurrency, so the result is in squence
		r := <-h.ResultCh
		So(r, ShouldNotBeNil)
		So(r.Err, ShouldBeNil)
		So(r.Resp.StatusCode, ShouldEqual, 200)
		So(r.Body, ShouldEqual, "1")

		r = <-h.ResultCh
		So(r, ShouldNotBeNil)
		So(r.Err, ShouldBeNil)
		So(r.Resp.StatusCode, ShouldEqual, 200)
		So(r.Body, ShouldEqual, "2")

		r = <-h.ResultCh
		So(r, ShouldNotBeNil)
		So(r.Err, ShouldBeNil)
		So(r.Resp.StatusCode, ShouldEqual, 200)
		So(r.Body, ShouldEqual, "3")
	})
}

func TestHead(t *testing.T) {
	h := newTestRespHandler(1)
	c := AsyncHttpClient{}

	Convey("Test Async Head without Concurrency limit", t, func() {
		c.Head("http://"+server+"/3", h.handle)

		r := <-h.ResultCh
		So(r, ShouldNotBeNil)
		So(r.Err, ShouldBeNil)
		So(r.Resp.StatusCode, ShouldEqual, 200)
	})
}

func TestDo(t *testing.T) {
	h := newTestRespHandler(1)
	c := AsyncHttpClient{}

	Convey("Test Async Do without Concurrency limit", t, func() {
		req, err := http.NewRequest("POST", "http://"+server+"/3", nil)
		So(err, ShouldBeNil)
		c.Do(req, h.handle)

		r := <-h.ResultCh
		So(r, ShouldNotBeNil)
		So(r.Err, ShouldBeNil)
		So(r.Resp.StatusCode, ShouldEqual, 200)
		So(r.Body, ShouldEqual, "3")
	})
}

func TestGetError(t *testing.T) {
	h := newTestRespHandler(1)
	c := AsyncHttpClient{}

	Convey("Test Async Get Error server", t, func() {
		foobar_server := "127.0.0.1:31713"
		c.Get("http://"+foobar_server+"/3", h.handle)

		r := <-h.ResultCh
		So(r, ShouldNotBeNil)
		So(r.Err, ShouldNotBeNil)
	})

	Convey("Test Async Get Error url", t, func() {
		c.Get("http://"+server+"/4", h.handle)

		r := <-h.ResultCh
		So(r, ShouldNotBeNil)
		So(r.Err, ShouldBeNil)
		So(r.Resp.StatusCode, ShouldEqual, 404)
	})
}

func TestCustomClient(t *testing.T) {
	h := newTestRespHandler(3)

	var myTransport http.RoundTripper = &http.Transport{
		ResponseHeaderTimeout: 200 * time.Millisecond,
	}

	client := &http.Client{Transport: myTransport}
	c := AsyncHttpClient{Client: client}

	Convey("Test Async Get using custom http.Client", t, func() {
		c.Get("http://"+server+"/1", h.handle)
		c.Get("http://"+server+"/2", h.handle)
		c.Get("http://"+server+"/3", h.handle)

		// Test 3 return asap
		r := <-h.ResultCh
		So(r, ShouldNotBeNil)
		So(r.Err, ShouldBeNil)
		So(r.Resp.StatusCode, ShouldEqual, 200)
		So(r.Body, ShouldEqual, "3")

		// Test 2 return after 150ms
		r = <-h.ResultCh
		So(r, ShouldNotBeNil)
		So(r.Err, ShouldBeNil)
		So(r.Resp.StatusCode, ShouldEqual, 200)
		So(r.Body, ShouldEqual, "2")

		// Test 1 return after 300ms, so it will timeout
		r = <-h.ResultCh
		So(r, ShouldNotBeNil)
		So(r.Err, ShouldNotBeNil)
	})
}

func TestGet(t *testing.T) {
	h := newTestRespHandler(3)
	c := AsyncHttpClient{}

	Convey("Test Async Get without Concurrency limit", t, func() {
		c.Get("http://"+server+"/1", h.handle)
		c.Get("http://"+server+"/2", h.handle)
		c.Get("http://"+server+"/3", h.handle)

		// Test 3 return asap
		r := <-h.ResultCh
		So(r, ShouldNotBeNil)
		So(r.Err, ShouldBeNil)
		So(r.Resp.StatusCode, ShouldEqual, 200)
		So(r.Body, ShouldEqual, "3")

		// Test 2 return after 150ms
		r = <-h.ResultCh
		So(r, ShouldNotBeNil)
		So(r.Err, ShouldBeNil)
		So(r.Resp.StatusCode, ShouldEqual, 200)
		So(r.Body, ShouldEqual, "2")

		// Test 1 return after 300ms
		r = <-h.ResultCh
		So(r, ShouldNotBeNil)
		So(r.Err, ShouldBeNil)
		So(r.Resp.StatusCode, ShouldEqual, 200)
		So(r.Body, ShouldEqual, "1")
	})
}

func TestOriginalClient(t *testing.T) {
	Convey("Test the server is ok", t, func() {
		Convey("Test 1 should finish in 300ms", func() {
			resp, err := http.Get("http://" + server + "/1")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.StatusCode, ShouldEqual, 200)

			body, err := getBody(resp)
			So(err, ShouldBeNil)
			So(body, ShouldEqual, "1")
		})

		Convey("Test 2 should finish in 150ms", func() {
			resp, err := http.Get("http://" + server + "/2")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.StatusCode, ShouldEqual, 200)

			body, err := getBody(resp)
			So(err, ShouldBeNil)
			So(body, ShouldEqual, "2")
		})

		Convey("Test 3 should finish asap", func() {
			resp, err := http.Get("http://" + server + "/3")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.StatusCode, ShouldEqual, 200)

			body, err := getBody(resp)
			So(err, ShouldBeNil)
			So(body, ShouldEqual, "3")
		})
	})
}

func init() {
	startServer()
}
