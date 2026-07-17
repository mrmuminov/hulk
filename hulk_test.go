package main

import (
	"bytes"
	"context"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- buildblock ---

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

// --- resolveAddr ---

func TestResolveAddr(t *testing.T) {
	tests := []struct {
		rawURL string
		want   string
	}{
		{"http://example.com", "example.com:80"},
		{"https://example.com", "example.com:443"},
		{"http://example.com:8080", "example.com:8080"},
		{"https://example.com:8443", "example.com:8443"},
	}

	for _, tt := range tests {
		u, _ := url.Parse(tt.rawURL)
		got := resolveAddr(u)
		if got != tt.want {
			t.Errorf("resolveAddr(%s) = %s, want %s", tt.rawURL, got, tt.want)
		}
	}
}

// --- extractPathInfo ---

func TestExtractPathInfo(t *testing.T) {
	tests := []struct {
		rawURL           string
		wantBasePath     string
		wantParamJoiner  string
	}{
		{"http://example.com", "/", "?"},
		{"http://example.com/path", "/path", "?"},
		{"http://example.com/?q=1", "/?q=1", "&"},
		{"http://example.com/path?q=1", "/path?q=1", "&"},
	}

	for _, tt := range tests {
		u, _ := url.Parse(tt.rawURL)
		basePath, joiner := extractPathInfo(u)
		if basePath != tt.wantBasePath || joiner != tt.wantParamJoiner {
			t.Errorf("extractPathInfo(%s) = (%q, %q), want (%q, %q)",
				tt.rawURL, basePath, joiner, tt.wantBasePath, tt.wantParamJoiner)
		}
	}
}

func TestExtractPathInfoWithQuery(t *testing.T) {
	u, _ := url.Parse("http://example.com/path?existing=1")
	basePath, joiner := extractPathInfo(u)

	if !strings.Contains(basePath, "existing=1") {
		t.Fatal("extractPathInfo should preserve existing query params")
	}
	if joiner != "&" {
		t.Fatal("with existing query, joiner should be &")
	}
}

// --- detectServerError ---

func TestDetectServerError(t *testing.T) {
	tests := []struct {
		resp string
		want bool
	}{
		{"HTTP/1.1 200 OK\r\n...", false},
		{"HTTP/1.1 404 Not Found\r\n...", false},
		{"HTTP/1.1 500 Internal Server Error\r\n...", true},
		{"HTTP/1.1 503 Service Unavailable\r\n...", true},
		{"HTTP/1.0 500 Internal Server Error\r\n...", true},
		{"HTTP/1.0 502 Bad Gateway\r\n...", true},
		{"", false},
	}

	for _, tt := range tests {
		got := detectServerError(tt.resp)
		if got != tt.want {
			t.Errorf("detectServerError(%q) = %v, want %v", tt.resp, got, tt.want)
		}
	}
}

// --- arrayFlags ---

func TestArrayFlags(t *testing.T) {
	var f arrayFlags

	if f.String() != "[]" {
		t.Fatalf("empty arrayFlags should be [], got %s", f.String())
	}

	f.Set("a")
	f.Set("b")
	f.Set("c")

	if f.String() != "[a,b,c]" {
		t.Fatalf("expected [a,b,c], got %s", f.String())
	}
}

// --- buildRequest ---

var testUAs = []string{"Mozilla/5.0 TestAgent"}
var testRefs = []string{"http://test.referer/?q="}

func TestBuildRequestGETMethod(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	var buf bytes.Buffer
	buildRequest(&buf, rng, "/", "?", "example.com", "", nil, testUAs, testRefs)

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

func TestBuildRequestPOSTMethod(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	var buf bytes.Buffer
	buildRequest(&buf, rng, "/submit", "", "test.local", "user=admin&pass=123", nil, testUAs, testRefs)

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
	buildRequest(&buf, rng, "/", "?", "x.com", "", custom, testUAs, testRefs)

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
	buildRequest(&buf, rng, "/path", "?", "x.com", "", nil, testUAs, testRefs)

	req := buf.String()

	if !strings.Contains(req, "/path?") {
		t.Fatal("GET request missing query param separator")
	}
}

func TestBuildRequestPOSTNoQueryParam(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	var buf bytes.Buffer
	buildRequest(&buf, rng, "/login", "", "x.com", "data=1", nil, testUAs, testRefs)

	req := buf.String()

	if strings.Contains(req, "/login?") {
		t.Fatal("POST should not have query param")
	}
}

func TestBuildRequestSelectedUserAgentPresent(t *testing.T) {
	rng := rand.New(rand.NewSource(6))
	var buf bytes.Buffer
	buildRequest(&buf, rng, "/", "?", "x.com", "", nil, testUAs, testRefs)

	req := buf.String()
	if !strings.Contains(req, "User-Agent: Mozilla/5.0 TestAgent\r\n") {
		t.Fatal("custom User-Agent should appear in request")
	}
}

func TestBuildRequestSelectedRefererPresent(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	var buf bytes.Buffer
	buildRequest(&buf, rng, "/", "?", "x.com", "", nil, testUAs, testRefs)

	req := buf.String()
	if !strings.Contains(req, "Referer: http://test.referer/?q=") {
		t.Fatal("custom Referer should appear in request")
	}
}

// --- run() ---

func testCtx() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	return ctx
}

func TestRunVersionFlag(t *testing.T) {
	code := run(context.Background(), []string{"-version"}, func(k string) string { return "" })
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestRunBadURL(t *testing.T) {
	code := run(context.Background(), []string{"-site", "://bad"}, func(k string) string { return "" })
	if code != 1 {
		t.Fatalf("expected exit code 1 for bad URL, got %d", code)
	}
}

func TestRunBadAgentsFile(t *testing.T) {
	code := run(context.Background(), []string{"-agents", "/nonexistent/file"}, func(k string) string { return "" })
	if code != 1 {
		t.Fatalf("expected exit code 1 for bad agents file, got %d", code)
	}
}

func TestRunHULKMAXPROCSZero(t *testing.T) {
	code := run(testCtx(), []string{}, func(k string) string {
		if k == "HULKMAXPROCS" {
			return "0"
		}
		return ""
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestRunExitsGracefully(t *testing.T) {
	code := run(testCtx(), []string{"-site", "http://localhost:1"}, func(k string) string { return "" })
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestRunLoadsAgentsFile(t *testing.T) {
	dir := t.TempDir()
	agentsFile := filepath.Join(dir, "agents.txt")
	os.WriteFile(agentsFile, []byte("CustomAgent/1.0\n"), 0644)

	code := run(testCtx(), []string{"-agents", agentsFile}, func(k string) string { return "" })
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}
