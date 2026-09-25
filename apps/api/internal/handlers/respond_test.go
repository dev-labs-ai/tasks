package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type errorBody struct {
	Error struct {
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields"`
	} `json:"error"`
}

// decodeHandler decodes {"title": ...} and echoes it back, the way a real
// handler would.
func decodeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Title string `json:"title"`
		}
		if err := DecodeJSON(w, r, &body); err != nil {
			WriteError(w, err)
			return
		}
		WriteData(w, http.StatusOK, body)
	})
}

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{"valid", `{"title":"Buy milk"}`, http.StatusOK, ""},
		{"not JSON", `title=Buy milk`, http.StatusBadRequest, CodeMalformedRequest},
		{"truncated", `{"title":`, http.StatusBadRequest, CodeMalformedRequest},
		{"empty body", ``, http.StatusBadRequest, CodeMalformedRequest},
		{"unknown field", `{"title":"x","admin":true}`, http.StatusBadRequest, CodeMalformedRequest},
		{"wrong type", `{"title":42}`, http.StatusBadRequest, CodeMalformedRequest},
		{"trailing value", `{"title":"x"}{"title":"y"}`, http.StatusBadRequest, CodeMalformedRequest},
		{"over 64 KiB", `{"title":"` + strings.Repeat("a", MaxBodyBytes) + `"}`, http.StatusRequestEntityTooLarge, CodePayloadTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			decodeHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body)))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if tt.wantCode == "" {
				return
			}
			if got := readError(t, rec).Error.Code; got != tt.wantCode {
				t.Errorf("code = %q, want %q", got, tt.wantCode)
			}
		})
	}
}

func TestWriteData(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteData(rec, http.StatusCreated, map[string]int{"id": 7})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"data":{"id":7}}` {
		t.Errorf("body = %s, want the data envelope", got)
	}
}

func TestWriteErrorIncludesFieldsOnlyWhenPresent(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, ValidationFailed(map[string]string{"title": "Title is required"}))

	body := readError(t, rec)
	if rec.Code != http.StatusBadRequest || body.Error.Code != CodeValidationFailed {
		t.Errorf("got %d %s, want 400 VALIDATION_FAILED", rec.Code, body.Error.Code)
	}
	if body.Error.Fields["title"] != "Title is required" {
		t.Errorf("fields = %v, want the title message", body.Error.Fields)
	}

	rec = httptest.NewRecorder()
	WriteError(rec, NotFound())
	if strings.Contains(rec.Body.String(), "fields") {
		t.Errorf("body = %s, want no fields key", rec.Body)
	}
}

func TestRouterAnswersUnknownRoutesWithNotFound(t *testing.T) {
	router := NewRouter(Deps{DB: fakePinger{}})

	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/v1/nothing-here", nil),
		httptest.NewRequest(http.MethodPost, "/healthz", nil),
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("%s %s: status = %d, want 404", req.Method, req.URL.Path, rec.Code)
		}
		if got := readError(t, rec).Error.Code; got != CodeNotFound {
			t.Errorf("%s %s: code = %q, want %q", req.Method, req.URL.Path, got, CodeNotFound)
		}
	}
}

func readError(t *testing.T, rec *httptest.ResponseRecorder) errorBody {
	t.Helper()
	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body, err)
	}
	return body
}
