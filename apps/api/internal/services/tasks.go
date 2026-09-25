package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dev-labs-ai/tasks/apps/api/internal/repositories"
)

var (
	// ErrTaskNotFound covers both a missing task and another user's task,
	// so the API never confirms that someone else's task exists (PRD FR-4).
	ErrTaskNotFound = errors.New("task not found")

	// ErrEmptyUpdate means an update named no field to change.
	ErrEmptyUpdate = errors.New("update has no fields")
)

type Task struct {
	ID        int64
	Title     string
	Completed bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TaskUpdate lists the fields to change; nil leaves a field as it is.
type TaskUpdate struct {
	Title     *string
	Completed *bool
}

// TaskService manages a user's tasks. Every method takes the user ID from
// the session and passes it to the query, which is what keeps users apart.
type TaskService struct {
	q repositories.Querier
}

func NewTaskService(q repositories.Querier) *TaskService {
	return &TaskService{q: q}
}

// List returns the user's tasks, newest first. status is the raw query
// parameter: empty, "pending" or "completed".
func (s *TaskService) List(ctx context.Context, userID int64, status string) ([]Task, error) {
	parsed, problem := ParseStatus(status)
	if problem != "" {
		return nil, &ValidationError{Fields: map[string]string{"status": problem}}
	}

	var completed pgtype.Bool
	switch parsed {
	case StatusPending:
		completed = pgtype.Bool{Bool: false, Valid: true}
	case StatusCompleted:
		completed = pgtype.Bool{Bool: true, Valid: true}
	}

	rows, err := s.q.ListTasks(ctx, repositories.ListTasksParams{UserID: userID, Completed: completed})
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	tasks := make([]Task, len(rows))
	for i, row := range rows {
		tasks[i] = toTask(row)
	}
	return tasks, nil
}

func (s *TaskService) Create(ctx context.Context, userID int64, title string) (Task, error) {
	title, problem := ValidateTitle(title)
	if problem != "" {
		return Task{}, &ValidationError{Fields: map[string]string{"title": problem}}
	}

	row, err := s.q.CreateTask(ctx, repositories.CreateTaskParams{UserID: userID, Title: title})
	if err != nil {
		return Task{}, fmt.Errorf("create task: %w", err)
	}
	return toTask(row), nil
}

func (s *TaskService) Update(ctx context.Context, userID, taskID int64, u TaskUpdate) (Task, error) {
	if u.Title == nil && u.Completed == nil {
		return Task{}, ErrEmptyUpdate
	}

	params := repositories.UpdateTaskParams{ID: taskID, UserID: userID}
	if u.Title != nil {
		title, problem := ValidateTitle(*u.Title)
		if problem != "" {
			return Task{}, &ValidationError{Fields: map[string]string{"title": problem}}
		}
		params.Title = pgtype.Text{String: title, Valid: true}
	}
	if u.Completed != nil {
		params.Completed = pgtype.Bool{Bool: *u.Completed, Valid: true}
	}

	row, err := s.q.UpdateTask(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, ErrTaskNotFound
	}
	if err != nil {
		return Task{}, fmt.Errorf("update task: %w", err)
	}
	return toTask(row), nil
}

func (s *TaskService) Delete(ctx context.Context, userID, taskID int64) error {
	n, err := s.q.DeleteTask(ctx, repositories.DeleteTaskParams{ID: taskID, UserID: userID})
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if n == 0 {
		return ErrTaskNotFound
	}
	return nil
}

func toTask(t repositories.Task) Task {
	return Task{ID: t.ID, Title: t.Title, Completed: t.Completed, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
}
