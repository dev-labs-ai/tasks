package handlers

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// signUp registers a user and returns their session cookie.
func (a *testAPI) signUp(email string) *http.Cookie {
	a.t.Helper()
	return sessionCookieOf(a.t, a.register(email, "correct horse"))
}

func (a *testAPI) createTask(session *http.Cookie, title string) taskResponse {
	a.t.Helper()
	rec := a.do(http.MethodPost, "/api/v1/tasks", `{"title":`+strconv.Quote(title)+`}`, session)
	if rec.Code != http.StatusCreated {
		a.t.Fatalf("create %q: status %d (body %s), want 201", title, rec.Code, rec.Body)
	}
	return decodeData[taskResponse](a.t, rec.Body.Bytes())
}

func (a *testAPI) listTasks(session *http.Cookie, query string) []taskResponse {
	a.t.Helper()
	rec := a.do(http.MethodGet, "/api/v1/tasks"+query, "", session)
	if rec.Code != http.StatusOK {
		a.t.Fatalf("list %q: status %d (body %s), want 200", query, rec.Code, rec.Body)
	}
	return decodeData[[]taskResponse](a.t, rec.Body.Bytes())
}

func taskPath(id int64) string {
	return "/api/v1/tasks/" + strconv.FormatInt(id, 10)
}

func decodeData[T any](t *testing.T, body []byte) T {
	t.Helper()
	var envelope struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return envelope.Data
}

func titles(tasks []taskResponse) []string {
	out := []string{}
	for _, t := range tasks {
		out = append(out, t.Title)
	}
	return out
}

func TestCreateTask(t *testing.T) {
	api := newTestAPI(t, false)
	session := api.signUp("ana@example.com")

	task := api.createTask(session, "  Buy milk ")
	if task.ID == 0 || task.Title != "Buy milk" || task.Completed || task.CreatedAt.IsZero() || task.UpdatedAt.IsZero() {
		t.Errorf("task = %+v, want an id, trimmed title, pending, timestamps", task)
	}

	for _, body := range []string{`{"title":""}`, `{"title":"   "}`, `{"title":` + strconv.Quote(strings.Repeat("a", 201)) + `}`, `{}`} {
		rec := api.do(http.MethodPost, "/api/v1/tasks", body, session)
		e := readError(t, rec)
		if rec.Code != http.StatusBadRequest || e.Error.Code != CodeValidationFailed || e.Error.Fields["title"] == "" {
			t.Errorf("create %s: %d %s, want 400 VALIDATION_FAILED on title", body, rec.Code, rec.Body)
		}
	}
}

func TestListTasksOrderAndFilter(t *testing.T) {
	api := newTestAPI(t, false)
	session := api.signUp("ana@example.com")

	first := api.createTask(session, "first")
	api.store.Now = api.store.Now.Add(time.Minute)
	api.createTask(session, "second")
	api.store.Now = api.store.Now.Add(time.Minute)
	api.createTask(session, "third")
	api.createTask(api.signUp("bob@example.com"), "Bob's")

	if rec := api.do(http.MethodPatch, taskPath(first.ID), `{"completed":true}`, session); rec.Code != http.StatusOK {
		t.Fatalf("complete: status %d", rec.Code)
	}

	for query, want := range map[string][]string{
		"":                  {"third", "second", "first"},
		"?status=pending":   {"third", "second"},
		"?status=completed": {"first"},
	} {
		if got := titles(api.listTasks(session, query)); !slices.Equal(got, want) {
			t.Errorf("list %q = %v, want %v", query, got, want)
		}
	}

	rec := api.do(http.MethodGet, "/api/v1/tasks?status=done", "", session)
	if e := readError(t, rec); rec.Code != http.StatusBadRequest || e.Error.Fields["status"] == "" {
		t.Errorf("invalid status: %d %s, want 400 VALIDATION_FAILED on status", rec.Code, rec.Body)
	}
}

func TestListTasksIsAnEmptyArrayNotNull(t *testing.T) {
	api := newTestAPI(t, false)
	rec := api.do(http.MethodGet, "/api/v1/tasks", "", api.signUp("ana@example.com"))

	if got := strings.TrimSpace(rec.Body.String()); got != `{"data":[]}` {
		t.Errorf("body = %s, want {\"data\":[]}", got)
	}
}

func TestUpdateTask(t *testing.T) {
	api := newTestAPI(t, false)
	session := api.signUp("ana@example.com")
	task := api.createTask(session, "Buy milk")

	patch := func(body string) (int, taskResponse) {
		rec := api.do(http.MethodPatch, taskPath(task.ID), body, session)
		if rec.Code != http.StatusOK {
			return rec.Code, taskResponse{}
		}
		return rec.Code, decodeData[taskResponse](t, rec.Body.Bytes())
	}

	if code, got := patch(`{"completed":true}`); code != http.StatusOK || !got.Completed {
		t.Errorf("complete: %d %+v, want 200 and completed", code, got)
	}
	if code, got := patch(`{"completed":false}`); code != http.StatusOK || got.Completed {
		t.Errorf("reopen: %d %+v, want 200 and pending", code, got)
	}
	if code, got := patch(`{"title":"Buy oat milk"}`); code != http.StatusOK || got.Title != "Buy oat milk" {
		t.Errorf("rename: %d %+v, want 200 and the new title", code, got)
	}

	rec := api.do(http.MethodPatch, taskPath(task.ID), `{"title":" "}`, session)
	if e := readError(t, rec); rec.Code != http.StatusBadRequest || e.Error.Fields["title"] == "" {
		t.Errorf("blank title: %d %s, want 400 VALIDATION_FAILED on title", rec.Code, rec.Body)
	}

	for _, body := range []string{`{}`, `{"completed":null}`} {
		rec := api.do(http.MethodPatch, taskPath(task.ID), body, session)
		if rec.Code != http.StatusBadRequest || readError(t, rec).Error.Code != CodeMalformedRequest {
			t.Errorf("patch %s: %d %s, want 400 MALFORMED_REQUEST", body, rec.Code, rec.Body)
		}
	}
}

func TestOtherUsersTasksAreNotFound(t *testing.T) {
	api := newTestAPI(t, false)
	ana := api.signUp("ana@example.com")
	bob := api.signUp("bob@example.com")
	task := api.createTask(ana, "Ana's task")

	for _, req := range []struct{ method, body string }{
		{http.MethodPatch, `{"completed":true}`},
		{http.MethodDelete, ""},
	} {
		rec := api.do(req.method, taskPath(task.ID), req.body, bob)
		if rec.Code != http.StatusNotFound || readError(t, rec).Error.Code != CodeTaskNotFound {
			t.Errorf("%s as Bob: %d %s, want 404 TASK_NOT_FOUND", req.method, rec.Code, rec.Body)
		}
	}

	if got := api.listTasks(ana, ""); len(got) != 1 || got[0].Completed {
		t.Errorf("Ana's tasks = %+v, want her task untouched", got)
	}
	if got := api.listTasks(bob, ""); len(got) != 0 {
		t.Errorf("Bob sees %+v, want no tasks", got)
	}
}

func TestDeleteTask(t *testing.T) {
	api := newTestAPI(t, false)
	session := api.signUp("ana@example.com")
	task := api.createTask(session, "Buy milk")

	if rec := api.do(http.MethodDelete, taskPath(task.ID), "", session); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("delete: %d %q, want 204 with no body", rec.Code, rec.Body)
	}
	rec := api.do(http.MethodDelete, taskPath(task.ID), "", session)
	if rec.Code != http.StatusNotFound || readError(t, rec).Error.Code != CodeTaskNotFound {
		t.Errorf("second delete: %d %s, want 404 TASK_NOT_FOUND", rec.Code, rec.Body)
	}
}

func TestTaskIDMustBeAPositiveInteger(t *testing.T) {
	api := newTestAPI(t, false)
	session := api.signUp("ana@example.com")

	for _, id := range []string{"abc", "0", "-1", "1.5", "99999999999999999999"} {
		rec := api.do(http.MethodDelete, "/api/v1/tasks/"+id, "", session)
		if rec.Code != http.StatusBadRequest || readError(t, rec).Error.Code != CodeMalformedRequest {
			t.Errorf("id %q: %d %s, want 400 MALFORMED_REQUEST", id, rec.Code, rec.Body)
		}
	}
}

func TestTaskRoutesNeedASession(t *testing.T) {
	api := newTestAPI(t, false)

	for _, req := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/tasks", ""},
		{http.MethodPost, "/api/v1/tasks", `{"title":"x"}`},
		{http.MethodPatch, "/api/v1/tasks/1", `{"completed":true}`},
		{http.MethodDelete, "/api/v1/tasks/1", ""},
	} {
		rec := api.do(req.method, req.path, req.body, nil)
		if rec.Code != http.StatusUnauthorized || readError(t, rec).Error.Code != CodeUnauthenticated {
			t.Errorf("%s %s without a session: %d %s, want 401 UNAUTHENTICATED", req.method, req.path, rec.Code, rec.Body)
		}
	}
}
