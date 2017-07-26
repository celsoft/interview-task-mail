package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_process(t *testing.T) {
	type testCase struct {
		Path     string
		PathType string
		Fail     bool
		Count    int
	}
	cases := []testCase{
		{"/go", "url", false, 2},
		{"/bad", "url", true, 0},
		{"/", "url", false, 0},
		{"/", "file", true, 0},
		{"main_test.go", "file", false, 2},
		{"main.go", "file", false, 0},
		{"notexist.go", "file", true, 0},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/go":
			// Count: 2
			io.WriteString(w, "<html><body>Go. Gophers. ConGo, gophra. Go language</body></html>")
		case "/bad":
			// Bad request
			w.WriteHeader(http.StatusBadRequest)
		default:
			// Count: 0
			io.WriteString(w, "Hello, gopher")
		}
	}))
	defer ts.Close()

	for _, test := range cases {
		var res result
		if test.PathType == "url" {
			log.Print("Start GET ", ts.URL, test.Path)
			res = process(ts.URL+test.Path, test.PathType)
		} else {
			log.Print("Read file ", test.Path)
			res = process(test.Path, test.PathType)
		}

		// Should fail and didn't
		if test.Fail && res.err == nil {
			log.Print("Fail\n")
			t.Fail()
			continue
		}

		// Should not fail and did
		if !test.Fail && res.err != nil {
			log.Print("Fail\n")
			t.Fail()
			continue
		}

		if test.Count != res.count {
			log.Print("Fail\n")
			t.Fail()
			continue
		}

		log.Print("Success\n")
	}
}
