package boomer

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestN(t *testing.T) {
	var count int64
	handler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, int64(1))
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	boomer := &Boomer{
		Request: req,
		N:       20,
		C:       2,
	}
	boomer.Run()
	if count != 20 {
		t.Errorf("Expected to boom 20 times, found %v", count)
	}
}

func TestQps(t *testing.T) {
	var wg sync.WaitGroup
	var count int64
	handler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, int64(1))
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	boomer := &Boomer{
		Request: req,
		N:       20,
		C:       2,
		Qps:     1,
	}
	wg.Add(1)
	time.AfterFunc(time.Second, func() {
		if count > 1 {
			t.Errorf("Expected to boom 1 times, found %v", count)
		}
		wg.Done()
	})
	go boomer.Run()
	wg.Wait()
}

func TestRequest(t *testing.T) {
	var uri, contentType, some, method, auth string
	handler := func(w http.ResponseWriter, r *http.Request) {
		uri = r.RequestURI
		method = r.Method
		contentType = r.Header.Get("Content-type")
		some = r.Header.Get("X-some")
		auth = r.Header.Get("Authorization")
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	header := make(http.Header)
	header.Add("Content-type", "text/html")
	header.Add("X-some", "value")
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header = header
	req.SetBasicAuth("username", "password")
	boomer := &Boomer{
		Request: req,
		N:       1,
		C:       1,
	}
	boomer.Run()
	if uri != "/" {
		t.Errorf("Uri is expected to be /, %v is found", uri)
	}
	if contentType != "text/html" {
		t.Errorf("Content type is expected to be text/html, %v is found", contentType)
	}
	if some != "value" {
		t.Errorf("X-some header is expected to be value, %v is found", some)
	}
	if auth != "Basic dXNlcm5hbWU6cGFzc3dvcmQ=" {
		t.Errorf("Basic authorization is not properly set")
	}
}

func TestBody(t *testing.T) {
	var count int64
	handler := func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		if string(body) == "Body" {
			atomic.AddInt64(&count, 1)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL, bytes.NewBuffer([]byte("Body")))
	boomer := &Boomer{
		Request:     req,
		RequestBody: "Body",
		N:           10,
		C:           1,
	}
	boomer.Run()
	if count != 10 {
		t.Errorf("Expected to boom 10 times, found %v", count)
	}
}
