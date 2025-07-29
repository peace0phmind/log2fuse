package log2fuse_test

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/peace0phmind/log2fuse"
)

type TestLoggerClock struct{}

func (c *TestLoggerClock) Now() time.Time {
	return time.Date(2020, time.December, 15, 13, 30, 40, 999999999, time.UTC)
}

type TestUUIDGenerator struct{}

func (g *TestUUIDGenerator) Generate() string {
	return "test-id"
}

type TestLogWriter struct {
	t        *testing.T
	expected string
}

func (w *TestLogWriter) Write(log string) error {
	w.t.Helper()
	if log != w.expected {
		w.t.Errorf("Expected: '%s', got: '%s'", w.expected, log)
	}
	return nil
}

// createContext creates text context with fake time and test log writer that assert the expected log.
func createContext(t *testing.T, expectedLog string) context.Context {
	t.Helper()
	clock := &TestLoggerClock{}
	uuidGenerator := &TestUUIDGenerator{}
	logWriter := &TestLogWriter{t: t, expected: expectedLog}
	return context.WithValue(context.WithValue(context.WithValue(context.Background(), log2fuse.LogWriterContextKey, logWriter), log2fuse.ClockContextKey, clock), log2fuse.UUIDGeneratorContextKey, uuidGenerator)
}

// doubleTheNumber reads the request, parses it as integer then returns its double.
// So the request and the response are not the same.
func doubleTheNumber(rw http.ResponseWriter, req *http.Request) {
	// Read the request body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer func(Body io.ReadCloser) {
		cerr := Body.Close()
		if cerr != nil {
			log.Printf("Failed to close reader: %v", cerr)
		}
	}(req.Body)

	// Parse the request body as an integer
	num, err := strconv.Atoi(string(body))
	if err != nil {
		http.Error(rw, "Bad Request: Request body must be an integer", http.StatusBadRequest)
		return
	}

	// Double the number
	result := num * 2

	// Write the result
	rw.WriteHeader(http.StatusOK)
	rw.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(rw, "%d", result)
}

// blackHole reads the request then it just returns HTTP OK without response body.
func blackHole(rw http.ResponseWriter, req *http.Request) {
	// Read the request body (to appear in logs)
	_, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer func(Body io.ReadCloser) {
		cerr := Body.Close()
		if cerr != nil {
			log.Printf("Failed to close reader: %v", cerr)
		}
	}(req.Body)
	rw.WriteHeader(http.StatusOK)
}

// alwaysError just returns HTTP 500
func alwaysError(rw http.ResponseWriter, req *http.Request) {
	http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
}

// alwaysFive does not read the request, just returns HTTP OK with response body 5.
func alwaysFive(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
	fmt.Fprintf(rw, "%d", 5)
}

// gzipAlwaysFive reads the request then returns GZip encoded HTTP OK with response body 5.
func gzipAlwaysFive(rw http.ResponseWriter, req *http.Request) {
	// Read the request body (to appear in logs)
	_, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer func(Body io.ReadCloser) {
		cerr := Body.Close()
		if cerr != nil {
			log.Printf("Failed to close reader: %v", cerr)
		}
	}(req.Body)

	// Return gzip response
	rw.Header().Set("Content-Encoding", "gzip")
	rw.WriteHeader(http.StatusOK)

	gz := gzip.NewWriter(rw)
	defer func() {
		if err := gz.Close(); err != nil {
			log.Printf("Failed to close gz: %v", err)
		}
	}()

	fmt.Fprintf(gz, "%d", 5)
}

func TestPost(t *testing.T) {
	expectedLogs := map[log2fuse.LogFormat]string{
		log2fuse.TextFormat: "127.0.0.1 POST /post: 200 OK HTTP/1.1\n\nRequest Headers:\nAccept: text/plain\nAuthorization: Bearer {\"alg\":\"HS256\",\"typ\":\"JWT\"}.{\"sub\":\"1234567890\",\"name\":\"John Doe\",\"iat\":1516239022}\n\nRequest Body:\n5\n\nResponse Headers:\nContent-Type: text/plain\n\nResponse Content Length: 2\n\nDuration: 0.000 ms\n\nResponse Body:\n10\n\n",
		log2fuse.JSONFormat: "{\"log.level\":\"info\",\"@timestamp\":\"2020-12-15T13:30:40.999Z\",\"message\":\"POST /post HTTP/1.1 200\",\"systemName\":\"HTTP\",\"remoteAddress\":\"127.0.0.1\",\"method\":\"POST\",\"path\":\"/post\",\"status\":200,\"statusText\":\"OK\",\"proto\":\"HTTP/1.1\",\"durationMs\":0,\"requestHeaders\":{\"Accept\":[\"text/plain\"],\"Authorization\":[\"Bearer {\\\"alg\\\":\\\"HS256\\\",\\\"typ\\\":\\\"JWT\\\"}.{\\\"sub\\\":\\\"1234567890\\\",\\\"name\\\":\\\"John Doe\\\",\\\"iat\\\":1516239022}\"]},\"requestBody\":\"5\",\"responseHeaders\":{\"Content-Type\":[\"text/plain\"]},\"responseContentLength\":2,\"responseBody\":\"10\",\"ecs.version\":\"1.6.0\",\"logId\":\"test-id\"}\n",
	}

	for logFormat, expectedLog := range expectedLogs {
		cfg := log2fuse.CreateConfig()
		cfg.LogFormat = logFormat
		cfg.JWTHeaders = []string{"Authorization"}

		ctx := createContext(t, expectedLog)

		handler, err := log2fuse.New(ctx, http.HandlerFunc(doubleTheNumber), cfg, "logger-plugin")
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/post", strings.NewReader("5"))
		if err != nil {
			t.Fatal(err)
		}
		req.RemoteAddr = "127.0.0.1"
		req.Header.Set("Accept", "text/plain")
		req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c")

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		// Check the response body
		if recorder.Body.String() != "10" {
			t.Errorf("Expected response body: '10', got: '%s'", recorder.Body.String())
		}
	}
}

func TestShortPost(t *testing.T) {
	expectedLogs := map[log2fuse.LogFormat]string{
		log2fuse.TextFormat: "127.0.0.1 POST /short-post: 200 OK HTTP/1.1\n\nRequest Headers:\nAccept: text/plain\nAuthorization: ██\n\nResponse Headers:\nContent-Type: text/plain\n\nResponse Content Length: 2\n\nDuration: 0.000 ms\n\n",
		log2fuse.JSONFormat: "{\"log.level\":\"info\",\"@timestamp\":\"2020-12-15T13:30:40.999Z\",\"message\":\"POST /short-post HTTP/1.1 200\",\"systemName\":\"HTTP\",\"remoteAddress\":\"127.0.0.1\",\"method\":\"POST\",\"path\":\"/short-post\",\"status\":200,\"statusText\":\"OK\",\"proto\":\"HTTP/1.1\",\"durationMs\":0,\"requestHeaders\":{\"Accept\":[\"text/plain\"],\"Authorization\":[\"██\"]},\"responseHeaders\":{\"Content-Type\":[\"text/plain\"]},\"responseContentLength\":2,\"ecs.version\":\"1.6.0\",\"logId\":\"test-id\"}\n",
	}

	cfgWithInterestedContentTypes := log2fuse.CreateConfig()
	cfgWithInterestedContentTypes.HeaderRedacts = []string{"Authorization"}
	cfgWithInterestedContentTypes.BodyContentTypes = []string{"text/html"}

	cfgWithBodyRedact := log2fuse.CreateConfig()
	cfgWithBodyRedact.HeaderRedacts = []string{"Authorization"}
	cfgWithBodyRedact.RequestBodyRedact = "POST /short-post"
	cfgWithBodyRedact.ResponseBodyRedact = "POST /short-post"

	configs := []*log2fuse.Config{cfgWithInterestedContentTypes, cfgWithBodyRedact}

	for _, cfg := range configs {
		for logFormat, expectedLog := range expectedLogs {
			cfg.LogFormat = logFormat

			ctx := createContext(t, expectedLog)

			handler, err := log2fuse.New(ctx, http.HandlerFunc(doubleTheNumber), cfg, "logger-plugin")
			if err != nil {
				t.Fatal(err)
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/short-post", strings.NewReader("5"))
			if err != nil {
				t.Fatal(err)
			}
			req.RemoteAddr = "127.0.0.1"
			req.Header.Set("Accept", "text/plain")
			req.Header.Set("Authorization", "secret")

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			// Check the response body
			if recorder.Body.String() != "10" {
				t.Errorf("Expected response body: '10', got: '%s'", recorder.Body.String())
			}
		}
	}
}

func TestEmptyPost(t *testing.T) {
	cfg := log2fuse.CreateConfig()

	ctx := createContext(t, "127.0.0.1 POST /empty-post: 200 OK HTTP/1.1\n\nRequest Body:\n5\n\nResponse Content Length: 0\n\nDuration: 0.000 ms\n\n")

	handler, err := log2fuse.New(ctx, http.HandlerFunc(blackHole), cfg, "logger-plugin")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/empty-post", strings.NewReader("5"))
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "127.0.0.1"

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Check the response body
	if recorder.Body.String() != "" {
		t.Errorf("Expected response body: '', got: '%s'", recorder.Body.String())
	}
}

func TestGet(t *testing.T) {
	cfg := log2fuse.CreateConfig()

	ctx := createContext(t, "127.0.0.1 GET /get: 200 OK HTTP/1.1\n\nRequest Headers:\nAccept: text/plain\n\nResponse Content Length: 1\n\nDuration: 0.000 ms\n\nResponse Body:\n5\n\n")

	handler, err := log2fuse.New(ctx, http.HandlerFunc(alwaysFive), cfg, "logger-plugin")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/get", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "127.0.0.1"
	req.Header.Set("Accept", "text/plain")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Check the response body
	if recorder.Body.String() != "5" {
		t.Errorf("Expected response body: '5', got: '%s'", recorder.Body.String())
	}
}

func TestPostGzipResponseWithRawRequest(t *testing.T) {
	cfg := log2fuse.CreateConfig()

	ctx := createContext(t, "127.0.0.1 POST /post: 200 OK HTTP/1.1\n\nRequest Headers:\nAccept: text/plain\n\nRequest Body:\nHello\n\nResponse Headers:\nContent-Encoding: gzip\n\nResponse Content Length: 25\n\nDuration: 0.000 ms\n\nResponse Body:\n5\n\n")

	handler, err := log2fuse.New(ctx, http.HandlerFunc(gzipAlwaysFive), cfg, "logger-plugin")
	if err != nil {
		t.Fatal(err)
	}

	reqBody := strings.NewReader("Hello")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/post", reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "127.0.0.1"
	req.Header.Set("Accept", "text/plain")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
}

func TestGetWithoutHeaders(t *testing.T) {
	cfg := log2fuse.CreateConfig()
	cfg.SilentHeaders = true

	ctx := createContext(t, "127.0.0.1 GET /get: 200 OK HTTP/1.1\n\nResponse Content Length: 1\n\nDuration: 0.000 ms\n\nResponse Body:\n5\n\n")

	handler, err := log2fuse.New(ctx, http.HandlerFunc(alwaysFive), cfg, "logger-plugin")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/get", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "127.0.0.1"
	req.Header.Set("Accept", "text/plain")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Check the response body
	if recorder.Body.String() != "5" {
		t.Errorf("Expected response body: '5', got: '%s'", recorder.Body.String())
	}
}

func TestGetWithoutLogID(t *testing.T) {
	ctx := createContext(t, "{\"log.level\":\"info\",\"@timestamp\":\"2020-12-15T13:30:40.999Z\",\"message\":\"GET /get-without-log-id HTTP/1.1 200\",\"systemName\":\"HTTP\",\"remoteAddress\":\"127.0.0.1\",\"method\":\"GET\",\"path\":\"/get-without-log-id\",\"status\":200,\"statusText\":\"OK\",\"proto\":\"HTTP/1.1\",\"durationMs\":0,\"responseContentLength\":1,\"responseBody\":\"5\",\"ecs.version\":\"1.6.0\"}\n")

	cfg := log2fuse.CreateConfig()
	cfg.LogFormat = log2fuse.JSONFormat
	cfg.GenerateLogID = false

	handler, err := log2fuse.New(ctx, http.HandlerFunc(alwaysFive), cfg, "logger-plugin")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/get-without-log-id", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "127.0.0.1"

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Check the response body
	if recorder.Body.String() != "5" {
		t.Errorf("Expected response body: '5', got: '%s'", recorder.Body.String())
	}
}

func TestGetError(t *testing.T) {
	cfg := log2fuse.CreateConfig()

	ctx := createContext(t, "127.0.0.1 GET /get-error: 500 Internal Server Error HTTP/1.1\n\nResponse Headers:\nContent-Type: text/plain; charset=utf-8\nX-Content-Type-Options: nosniff\n\nResponse Content Length: 22\n\nDuration: 0.000 ms\n\nResponse Body:\nInternal Server Error\n\n\n")

	handler, err := log2fuse.New(ctx, http.HandlerFunc(alwaysError), cfg, "logger-plugin")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/get-error", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "127.0.0.1"

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Check the response body
	if recorder.Body.String() != "Internal Server Error\n" {
		t.Errorf("Expected response body: 'Internal Server Error\n', got: '%s'", recorder.Body.String())
	}
}

func TestGetWebsocket(t *testing.T) {
	cfg := log2fuse.CreateConfig()

	ctx := createContext(t, "LogWriter should not have been called")

	handler, err := log2fuse.New(ctx, http.HandlerFunc(alwaysFive), cfg, "logger-plugin")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/get-websocket", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "127.0.0.1"
	req.Header.Set("Upgrade", "websocket")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Check the response body
	if recorder.Body.String() != "5" {
		t.Errorf("Expected response body: '5', got: '%s'", recorder.Body.String())
	}
}

func TestEmptyGet(t *testing.T) {
	cfg := log2fuse.CreateConfig()

	ctx := createContext(t, "127.0.0.1 GET /empty-get: 200 OK HTTP/1.1\n\nResponse Content Length: 0\n\nDuration: 0.000 ms\n\n")
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})

	handler, err := log2fuse.New(ctx, next, cfg, "logger-plugin")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/empty-get", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "127.0.0.1"

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Check the response body
	if recorder.Body.String() != "" {
		t.Errorf("Expected response body: '', got: '%s'", recorder.Body.String())
	}
}

func TestDisabled(t *testing.T) {
	cfg := log2fuse.CreateConfig()
	cfg.Enabled = false

	ctx := createContext(t, "LogWriter should not have been called")

	handler, err := log2fuse.New(ctx, http.HandlerFunc(alwaysFive), cfg, "logger-plugin")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/disabled", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "127.0.0.1"

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Check the response body
	if recorder.Body.String() != "5" {
		t.Errorf("Expected response body: '5', got: '%s'", recorder.Body.String())
	}
}
