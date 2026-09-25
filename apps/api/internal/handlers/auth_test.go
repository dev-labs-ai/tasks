package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dev-labs-ai/tasks/apps/api/internal/services"
	"github.com/dev-labs-ai/tasks/apps/api/internal/services/servicestest"
)

// testAPI runs the real router and services over the in-memory store.
type testAPI struct {
	t      *testing.T
	router http.Handler
	store  *servicestest.Store
	// bodies collects every response body, to check none leaks a hash.
	bodies []string
}

func newTestAPI(t *testing.T, secureCookies bool) *testAPI {
	t.Helper()
	store := servicestest.NewStore(time.Now())
	router := NewRouter(Deps{
		DB:            fakePinger{},
		Auth:          services.NewAuthService(store, &servicestest.Passwords{}),
		Tasks:         services.NewTaskService(store),
		SecureCookies: secureCookies,
	})
	api := &testAPI{t: t, router: router, store: store}
	t.Cleanup(api.assertNoHashLeaked)
	return api
}

// do sends a request, with the session cookie when one is given.
func (a *testAPI) do(method, path, body string, session *http.Cookie) *httptest.ResponseRecorder {
	a.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if session != nil {
		req.AddCookie(session)
	}
	rec := httptest.NewRecorder()
	a.router.ServeHTTP(rec, req)
	a.bodies = append(a.bodies, rec.Body.String())
	return rec
}

func (a *testAPI) register(email, password string) *httptest.ResponseRecorder {
	a.t.Helper()
	return a.do(http.MethodPost, "/api/v1/auth/register", credentialsJSON(email, password), nil)
}

func (a *testAPI) login(email, password string) *httptest.ResponseRecorder {
	a.t.Helper()
	return a.do(http.MethodPost, "/api/v1/auth/login", credentialsJSON(email, password), nil)
}

// assertNoHashLeaked looks for the fake hashes ("hash:" + password) and
// for hash field names in every response.
func (a *testAPI) assertNoHashLeaked() {
	for _, body := range a.bodies {
		for _, leak := range []string{"hash:", "password_hash", "passwordHash", "PasswordHash"} {
			if strings.Contains(body, leak) {
				a.t.Errorf("response body leaks %q: %s", leak, body)
			}
		}
	}
}

func credentialsJSON(email, password string) string {
	b, _ := json.Marshal(credentials{Email: email, Password: password})
	return string(b)
}

func sessionCookieOf(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == SessionCookie {
			return c
		}
	}
	t.Fatalf("response sets no %q cookie; Set-Cookie = %q", SessionCookie, rec.Header().Values("Set-Cookie"))
	return nil
}

func TestRegisterSetsTheSessionCookie(t *testing.T) {
	for _, secure := range []bool{false, true} {
		t.Run("secure="+strconv.FormatBool(secure), func(t *testing.T) {
			api := newTestAPI(t, secure)

			rec := api.register("Ana@Example.com", "correct horse")

			if rec.Code != http.StatusCreated {
				t.Fatalf("status = %d, want 201 (body %s)", rec.Code, rec.Body)
			}
			c := sessionCookieOf(t, rec)
			if c.Value == "" || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.MaxAge != 86400 {
				t.Errorf("cookie = %+v, want a value, HttpOnly, SameSite=Lax, Path=/, Max-Age=86400", c)
			}
			if c.Secure != secure {
				t.Errorf("Secure = %v, want %v", c.Secure, secure)
			}

			var body struct {
				Data userResponse `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Data.ID == 0 || body.Data.Email != "ana@example.com" || body.Data.CreatedAt.IsZero() {
				t.Errorf("user = %+v, want id, normalized email and createdAt", body.Data)
			}
		})
	}
}

func TestRegisterErrors(t *testing.T) {
	api := newTestAPI(t, false)
	if rec := api.register("ana@example.com", "correct horse"); rec.Code != http.StatusCreated {
		t.Fatalf("first register: status %d", rec.Code)
	}

	rec := api.register("ANA@example.com", "another password")
	if rec.Code != http.StatusConflict || readError(t, rec).Error.Code != CodeEmailTaken {
		t.Errorf("duplicate: %d %s, want 409 EMAIL_TAKEN", rec.Code, rec.Body)
	}

	rec = api.register("Ana <ana@example.com>", "short")
	body := readError(t, rec)
	if rec.Code != http.StatusBadRequest || body.Error.Code != CodeValidationFailed {
		t.Fatalf("invalid fields: %d %s, want 400 VALIDATION_FAILED", rec.Code, rec.Body)
	}
	if body.Error.Fields["email"] == "" || body.Error.Fields["password"] == "" {
		t.Errorf("fields = %v, want messages for email and password", body.Error.Fields)
	}
}

func TestLoginFailuresAreIndistinguishable(t *testing.T) {
	api := newTestAPI(t, false)
	api.register("ana@example.com", "correct horse")

	unknown := api.login("nobody@example.com", "correct horse")
	wrong := api.login("ana@example.com", "wrong password")

	if unknown.Code != http.StatusUnauthorized || wrong.Code != http.StatusUnauthorized {
		t.Fatalf("statuses = %d and %d, want 401 for both", unknown.Code, wrong.Code)
	}
	if unknown.Body.String() != wrong.Body.String() {
		t.Errorf("bodies differ:\nunknown email:  %s\nwrong password: %s", unknown.Body, wrong.Body)
	}
	if code := readError(t, wrong).Error.Code; code != CodeInvalidCredentials {
		t.Errorf("code = %q, want %q", code, CodeInvalidCredentials)
	}
}

func TestLoginSucceeds(t *testing.T) {
	api := newTestAPI(t, false)
	api.register("ana@example.com", "correct horse")

	rec := api.login("ana@example.com", "correct horse")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body)
	}
	if me := api.do(http.MethodGet, "/api/v1/auth/me", "", sessionCookieOf(t, rec)); me.Code != http.StatusOK {
		t.Errorf("me with the login cookie: status %d, want 200", me.Code)
	}
}

func TestSixthLoginIsRefusedWithRetryAfter(t *testing.T) {
	api := newTestAPI(t, false)
	api.register("ana@example.com", "correct horse")

	for i := 1; i <= services.MaxLoginAttempts; i++ {
		if rec := api.login("ana@example.com", "wrong password"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status %d, want 401", i, rec.Code)
		}
	}
	rec := api.login("ana@example.com", "correct horse")

	if rec.Code != http.StatusTooManyRequests || readError(t, rec).Error.Code != CodeTooManyAttempts {
		t.Fatalf("6th attempt: %d %s, want 429 TOO_MANY_ATTEMPTS", rec.Code, rec.Body)
	}
	seconds, err := strconv.Atoi(rec.Header().Get("Retry-After"))
	if err != nil || seconds < 1 || seconds > 15*60 {
		t.Errorf("Retry-After = %q, want whole seconds up to 900", rec.Header().Get("Retry-After"))
	}
}

func TestLogoutEndsTheSession(t *testing.T) {
	api := newTestAPI(t, false)
	session := sessionCookieOf(t, api.register("ana@example.com", "correct horse"))

	if rec := api.do(http.MethodGet, "/api/v1/auth/me", "", session); rec.Code != http.StatusOK {
		t.Fatalf("me before logout: status %d, want 200", rec.Code)
	}

	rec := api.do(http.MethodPost, "/api/v1/auth/logout", "", session)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout: status %d, want 204", rec.Code)
	}
	if cleared := sessionCookieOf(t, rec); cleared.MaxAge >= 0 || cleared.Value != "" {
		t.Errorf("logout cookie = %+v, want an expired, empty cookie", cleared)
	}

	rec = api.do(http.MethodGet, "/api/v1/auth/me", "", session)
	if rec.Code != http.StatusUnauthorized || readError(t, rec).Error.Code != CodeUnauthenticated {
		t.Errorf("me after logout: %d %s, want 401 UNAUTHENTICATED", rec.Code, rec.Body)
	}
}

func TestAuthenticatedRoutesNeedASession(t *testing.T) {
	api := newTestAPI(t, false)

	for _, req := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/auth/me"},
		{http.MethodPost, "/api/v1/auth/logout"},
	} {
		for name, cookie := range map[string]*http.Cookie{
			"no cookie":     nil,
			"unknown token": {Name: SessionCookie, Value: "forged"},
		} {
			rec := api.do(req.method, req.path, "", cookie)
			if rec.Code != http.StatusUnauthorized || readError(t, rec).Error.Code != CodeUnauthenticated {
				t.Errorf("%s %s with %s: %d %s, want 401 UNAUTHENTICATED", req.method, req.path, name, rec.Code, rec.Body)
			}
		}
	}
}
