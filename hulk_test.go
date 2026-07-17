package main

import (
	"bytes"
	"math/rand"
	"strings"
	"testing"
)

func TestBuildblockSize(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for size := 0; size < 100; size++ {
		s := buildblock(rng, size)
		if len(s) != size {
			t.Fatalf("buildblock(%d) returned length %d", size, len(s))
		}
	}
}

func TestBuildblockCharset(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	for i := 0; i < 1000; i++ {
		s := buildblock(rng, 50)
		for _, c := range s {
			if c < 'A' || c > 'Z' {
				t.Fatalf("buildblock returned char %c (not A-Z)", c)
			}
		}
	}
}

func TestBuildblockDeterministic(t *testing.T) {
	rng1 := rand.New(rand.NewSource(0))
	rng2 := rand.New(rand.NewSource(0))
	s1 := buildblock(rng1, 10)
	s2 := buildblock(rng2, 10)
	if s1 != s2 {
		t.Fatal("same seed should produce same output")
	}
}

func TestBuildRequestGET(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	var buf bytes.Buffer
	buildRequest(&buf, rng, "/", "?", "example.com", "", nil)

	req := buf.String()

	if !strings.HasPrefix(req, "GET ") {
		t.Fatal("expected GET method")
	}
	if !strings.Contains(req, " HTTP/1.1\r\n") {
		t.Fatal("missing HTTP/1.1")
	}
	if !strings.Contains(req, "Host: example.com\r\n") {
		t.Fatal("missing Host header")
	}
	if strings.Contains(req, "Content-Length") {
		t.Fatal("GET should not have Content-Length")
	}
	if !strings.Contains(req, "\r\n\r\n") {
		t.Fatal("missing end of headers")
	}
}

func TestBuildRequestPOST(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	var buf bytes.Buffer
	buildRequest(&buf, rng, "/submit", "", "test.local", "user=admin&pass=123", nil)

	req := buf.String()

	if !strings.HasPrefix(req, "POST ") {
		t.Fatal("expected POST method")
	}
	if !strings.Contains(req, "Content-Length: 19\r\n") {
		t.Fatal("missing or wrong Content-Length")
	}
	if !strings.HasSuffix(strings.TrimSpace(req), "user=admin&pass=123") {
		t.Fatal("missing POST body")
	}
}

func TestBuildRequestCustomHeaders(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	var buf bytes.Buffer
	custom := []string{"X-Test: hello", "X-Foo: bar"}
	buildRequest(&buf, rng, "/", "?", "x.com", "", custom)

	req := buf.String()

	if !strings.Contains(req, "X-Test: hello\r\n") {
		t.Fatal("missing custom header X-Test")
	}
	if !strings.Contains(req, "X-Foo: bar\r\n") {
		t.Fatal("missing custom header X-Foo")
	}
}

func TestBuildRequestGETHasQueryParam(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	var buf bytes.Buffer
	buildRequest(&buf, rng, "/path", "?", "x.com", "", nil)

	req := buf.String()

	if !strings.Contains(req, "/path?") {
		t.Fatal("GET request missing query param separator")
	}
}

func TestBuildRequestPOSTNoQueryParam(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	var buf bytes.Buffer
	buildRequest(&buf, rng, "/login", "", "x.com", "data=1", nil)

	req := buf.String()

	if strings.Contains(req, "/login?") {
		t.Fatal("POST should not have query param")
	}
}
