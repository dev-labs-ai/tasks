package services

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/dev-labs-ai/tasks/apps/api/internal/services/servicestest"
)

const (
	ana int64 = 1
	bob int64 = 2
)

func newTaskService() (*TaskService, *servicestest.Store) {
	store := servicestest.NewStore(testNow)
	return NewTaskService(store), store
}

func ptr[T any](v T) *T { return &v }

func wantFieldError(t *testing.T, err error, field string) {
	t.Helper()
	ve, ok := errors.AsType[*ValidationError](err)
	if !ok || ve.Fields[field] == "" {
		t.Errorf("err = %v, want a *ValidationError for %q", err, field)
	}
}

func TestCreateTask(t *testing.T) {
	s, _ := newTaskService()

	task, err := s.Create(t.Context(), ana, "  Buy milk  ")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if task.Title != "Buy milk" || task.Completed {
		t.Errorf("task = %+v, want trimmed title, pending", task)
	}

	for _, title := range []string{"", "   ", strings.Repeat("😀", 201)} {
		_, err := s.Create(t.Context(), ana, title)
		wantFieldError(t, err, "title")
	}
}

func TestListTasks(t *testing.T) {
	s, store := newTaskService()
	first, _ := s.Create(t.Context(), ana, "first")
	store.Now = store.Now.Add(1)
	second, _ := s.Create(t.Context(), ana, "second")
	if _, err := s.Create(t.Context(), bob, "Bob's"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := s.Update(t.Context(), ana, first.ID, TaskUpdate{Completed: ptr(true)}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	tests := []struct {
		status string
		want   []int64
	}{
		{"", []int64{second.ID, first.ID}},
		{"pending", []int64{second.ID}},
		{"completed", []int64{first.ID}},
	}
	for _, tt := range tests {
		t.Run("status="+tt.status, func(t *testing.T) {
			tasks, err := s.List(t.Context(), ana, tt.status)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			var got []int64
			for _, task := range tasks {
				got = append(got, task.ID)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("ids = %v, want %v", got, tt.want)
			}
		})
	}

	_, err := s.List(t.Context(), ana, "done")
	wantFieldError(t, err, "status")
}

func TestUpdateTask(t *testing.T) {
	s, _ := newTaskService()
	task, _ := s.Create(t.Context(), ana, "Buy milk")

	renamed, err := s.Update(t.Context(), ana, task.ID, TaskUpdate{Title: ptr("  Buy oat milk ")})
	if err != nil || renamed.Title != "Buy oat milk" || renamed.Completed {
		t.Errorf("rename: %+v, %v; want trimmed new title, still pending", renamed, err)
	}

	done, err := s.Update(t.Context(), ana, task.ID, TaskUpdate{Completed: ptr(true)})
	if err != nil || !done.Completed || done.Title != "Buy oat milk" {
		t.Errorf("complete: %+v, %v; want completed, title kept", done, err)
	}

	reopened, err := s.Update(t.Context(), ana, task.ID, TaskUpdate{Completed: ptr(false)})
	if err != nil || reopened.Completed {
		t.Errorf("reopen: %+v, %v; want pending", reopened, err)
	}

	_, err = s.Update(t.Context(), ana, task.ID, TaskUpdate{Title: ptr(" ")})
	wantFieldError(t, err, "title")

	if _, err := s.Update(t.Context(), ana, task.ID, TaskUpdate{}); !errors.Is(err, ErrEmptyUpdate) {
		t.Errorf("empty update: err = %v, want ErrEmptyUpdate", err)
	}
}

func TestUpdateAndDeleteReportOtherUsersTasksAsNotFound(t *testing.T) {
	s, store := newTaskService()
	task, _ := s.Create(t.Context(), ana, "Ana's task")

	if _, err := s.Update(t.Context(), bob, task.ID, TaskUpdate{Completed: ptr(true)}); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("update as Bob: err = %v, want ErrTaskNotFound", err)
	}
	if err := s.Delete(t.Context(), bob, task.ID); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("delete as Bob: err = %v, want ErrTaskNotFound", err)
	}
	if store.Tasks[task.ID].Completed {
		t.Error("Bob's update changed Ana's task")
	}

	if _, err := s.Update(t.Context(), ana, 999, TaskUpdate{Completed: ptr(true)}); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("update missing task: err = %v, want ErrTaskNotFound", err)
	}

	if err := s.Delete(t.Context(), ana, task.ID); err != nil {
		t.Fatalf("delete as Ana: %v", err)
	}
	if err := s.Delete(t.Context(), ana, task.ID); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("second delete: err = %v, want ErrTaskNotFound", err)
	}
}
