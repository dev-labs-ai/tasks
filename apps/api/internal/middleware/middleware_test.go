package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dev-labs-ai/tasks/apps/api/internal/handlers"
)

var ok = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
})

func TestRecoverHidesPanicDetails(t *testing.T) {
	logs := captureLogs(t)
	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("secret: connection string postgres://user:pass@db")
	})

	rec := httptest.NewRecorder()
	Recover(panicking).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if got := errorCode(t, rec); got != handlers.CodeInternalError {
		t.Errorf("code = %q, want %q", got, handlers.CodeInternalError)
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("response leaks the panic value: %s", rec.Body)
	}
	if !strings.Contains(logs.String(), "secret") {
		t.Error("the panic value is missing from the logs")
	}
}

func TestCrossOrigin(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		headers    map[string]string
		wantStatus int
	}{
		{"cross-site POST", http.MethodPost, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"POST from another origin, no fetch metadata", http.MethodPost, map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden},
		{"same-origin POST", http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-origin"}, http.StatusNoContent},
		{"POST without browser headers", http.MethodPost, nil, http.StatusNoContent},
		{"cross-site GET", http.MethodGet, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://tasks.example/api/v1/tasks", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			CrossOrigin(ok).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusForbidden {
				if got := errorCode(t, rec); got != handlers.CodeForbiddenOrigin {
					t.Errorf("code = %q, want %q", got, handlers.CodeForbiddenOrigin)
				}
			}
		})
	}
}

func TestLogRecordsStatusWithoutQueryOrHeaders(t *testing.T) {
	logs := captureLogs(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?status=pending", nil)
	req.Header.Set("Cookie", "session=secret-token")
	Log(ok).ServeHTTP(httptest.NewRecorder(), req)

	line := logs.String()
	for _, want := range []string{"method=GET", "path=/api/v1/tasks", "status=204"} {
		if !strings.Contains(line, want) {
			t.Errorf("log %q is missing %q", line, want)
		}
	}
	for _, leak := range []string{"secret-token", "status=pending"} {
		if strings.Contains(line, leak) {
			t.Errorf("log %q contains %q", line, leak)
		}
	}
}

func TestLogSeesTheStatusRecoverWrites(t *testing.T) {
	logs := captureLogs(t)
	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })

	Log(Recover(panicking)).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !strings.Contains(logs.String(), "status=500") {
		t.Errorf("request log %q does not record the 500", logs)
	}
}

// captureLogs sends slog's default logger to a buffer for one test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body, err)
	}
	return body.Error.Code
}
