package services

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/dev-labs-ai/tasks/apps/api/internal/repositories"
)

// fakeStore keeps rows in memory and mirrors the SQL semantics the
// services rely on: ownership filters, unique emails and "no rows" errors.
// The SQL itself is covered by the integration tests.
type fakeStore struct {
	now      time.Time
	nextID   int64
	users    map[int64]repositories.User
	sessions map[string]repositories.Session
	attempts map[string]int32
	tasks    map[int64]repositories.Task
	calls    map[string]int
}

var windowEnd = time.Date(2026, 9, 25, 12, 15, 0, 0, time.UTC)

func newFakeStore(now time.Time) *fakeStore {
	return &fakeStore{
		now:      now,
		users:    map[int64]repositories.User{},
		sessions: map[string]repositories.Session{},
		attempts: map[string]int32{},
		tasks:    map[int64]repositories.Task{},
		calls:    map[string]int{},
	}
}

func (f *fakeStore) id() int64 {
	f.nextID++
	return f.nextID
}

func (f *fakeStore) InTx(_ context.Context, fn func(repositories.Querier) error) error {
	f.calls["InTx"]++
	return fn(f)
}

func (f *fakeStore) CreateUser(_ context.Context, arg repositories.CreateUserParams) (repositories.User, error) {
	f.calls["CreateUser"]++
	for _, u := range f.users {
		if u.Email == arg.Email {
			return repositories.User{}, &pgconn.PgError{Code: "23505"}
		}
	}
	u := repositories.User{ID: f.id(), Email: arg.Email, PasswordHash: arg.PasswordHash, CreatedAt: f.now}
	f.users[u.ID] = u
	return u, nil
}

func (f *fakeStore) GetUserByEmail(_ context.Context, email string) (repositories.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return repositories.User{}, pgx.ErrNoRows
}

func (f *fakeStore) GetUserByID(_ context.Context, id int64) (repositories.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return repositories.User{}, pgx.ErrNoRows
}

func (f *fakeStore) CreateSession(_ context.Context, arg repositories.CreateSessionParams) error {
	f.calls["CreateSession"]++
	f.sessions[string(arg.TokenHash)] = repositories.Session{
		TokenHash: arg.TokenHash, UserID: arg.UserID, CreatedAt: f.now, ExpiresAt: arg.ExpiresAt,
	}
	return nil
}

func (f *fakeStore) GetSessionUserID(_ context.Context, tokenHash []byte) (int64, error) {
	s, ok := f.sessions[string(tokenHash)]
	if !ok || !s.ExpiresAt.After(f.now) {
		return 0, pgx.ErrNoRows
	}
	return s.UserID, nil
}

func (f *fakeStore) DeleteSession(_ context.Context, tokenHash []byte) error {
	delete(f.sessions, string(tokenHash))
	return nil
}

func (f *fakeStore) DeleteExpiredSessions(context.Context) (int64, error) {
	var n int64
	for k, s := range f.sessions {
		if !s.ExpiresAt.After(f.now) {
			delete(f.sessions, k)
			n++
		}
	}
	return n, nil
}

func (f *fakeStore) RegisterLoginAttempt(_ context.Context, email string) (repositories.RegisterLoginAttemptRow, error) {
	f.calls["RegisterLoginAttempt"]++
	f.attempts[email]++
	return repositories.RegisterLoginAttemptRow{Attempts: f.attempts[email], WindowEnd: windowEnd}, nil
}

func (f *fakeStore) ClearLoginAttempts(_ context.Context, email string) error {
	f.calls["ClearLoginAttempts"]++
	delete(f.attempts, email)
	return nil
}

func (f *fakeStore) DeleteStaleLoginAttempts(context.Context) (int64, error) {
	return 0, nil
}

func (f *fakeStore) CreateTask(_ context.Context, arg repositories.CreateTaskParams) (repositories.Task, error) {
	t := repositories.Task{ID: f.id(), UserID: arg.UserID, Title: arg.Title, CreatedAt: f.now, UpdatedAt: f.now}
	f.tasks[t.ID] = t
	return t, nil
}

func (f *fakeStore) ListTasks(_ context.Context, arg repositories.ListTasksParams) ([]repositories.Task, error) {
	out := []repositories.Task{}
	for _, t := range f.tasks {
		if t.UserID != arg.UserID || (arg.Completed.Valid && t.Completed != arg.Completed.Bool) {
			continue
		}
		out = append(out, t)
	}
	slices.SortFunc(out, func(a, b repositories.Task) int {
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID)
	})
	return out, nil
}

func (f *fakeStore) UpdateTask(_ context.Context, arg repositories.UpdateTaskParams) (repositories.Task, error) {
	t, ok := f.tasks[arg.ID]
	if !ok || t.UserID != arg.UserID {
		return repositories.Task{}, pgx.ErrNoRows
	}
	if arg.Title.Valid {
		t.Title = arg.Title.String
	}
	if arg.Completed.Valid {
		t.Completed = arg.Completed.Bool
	}
	t.UpdatedAt = f.now
	f.tasks[t.ID] = t
	return t, nil
}

func (f *fakeStore) DeleteTask(_ context.Context, arg repositories.DeleteTaskParams) (int64, error) {
	t, ok := f.tasks[arg.ID]
	if !ok || t.UserID != arg.UserID {
		return 0, nil
	}
	delete(f.tasks, arg.ID)
	return 1, nil
}

// fakePasswords stores "hash:" + password and counts calls, so tests can
// tell whether a password was checked at all.
type fakePasswords struct {
	hashes, verifies, dummies int
}

func (p *fakePasswords) Hash(password string) string {
	p.hashes++
	return "hash:" + password
}

func (p *fakePasswords) Verify(password, encoded string) (bool, error) {
	p.verifies++
	return encoded == "hash:"+password, nil
}

func (p *fakePasswords) VerifyDummy(string) {
	p.dummies++
}
