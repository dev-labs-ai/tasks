// Package servicestest provides in-memory stand-ins for the services'
// dependencies, for unit and HTTP tests that should not need PostgreSQL.
package servicestest

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/dev-labs-ai/tasks/apps/api/internal/repositories"
)

// Store keeps rows in memory and mirrors the SQL semantics the services
// rely on: ownership filters, unique emails and "no rows" errors. The SQL
// itself is covered by the integration tests. It is not safe for
// concurrent use.
type Store struct {
	// Now is the database clock: it stamps rows and decides expiry.
	Now time.Time
	// WindowEnd is what RegisterLoginAttempt reports as the end of the
	// login window.
	WindowEnd time.Time

	Users    map[int64]repositories.User
	Sessions map[string]repositories.Session // keyed by string(token hash)
	Attempts map[string]int32
	Tasks    map[int64]repositories.Task
	// Calls counts calls to the methods tests make assertions about.
	Calls map[string]int

	nextID int64
}

func NewStore(now time.Time) *Store {
	return &Store{
		Now:       now,
		WindowEnd: now.Add(15 * time.Minute),
		Users:     map[int64]repositories.User{},
		Sessions:  map[string]repositories.Session{},
		Attempts:  map[string]int32{},
		Tasks:     map[int64]repositories.Task{},
		Calls:     map[string]int{},
	}
}

func (f *Store) id() int64 {
	f.nextID++
	return f.nextID
}

func (f *Store) InTx(_ context.Context, fn func(repositories.Querier) error) error {
	f.Calls["InTx"]++
	return fn(f)
}

func (f *Store) CreateUser(_ context.Context, arg repositories.CreateUserParams) (repositories.User, error) {
	f.Calls["CreateUser"]++
	for _, u := range f.Users {
		if u.Email == arg.Email {
			return repositories.User{}, &pgconn.PgError{Code: "23505"}
		}
	}
	u := repositories.User{ID: f.id(), Email: arg.Email, PasswordHash: arg.PasswordHash, CreatedAt: f.Now}
	f.Users[u.ID] = u
	return u, nil
}

func (f *Store) GetUserByEmail(_ context.Context, email string) (repositories.User, error) {
	for _, u := range f.Users {
		if u.Email == email {
			return u, nil
		}
	}
	return repositories.User{}, pgx.ErrNoRows
}

func (f *Store) GetUserByID(_ context.Context, id int64) (repositories.User, error) {
	if u, ok := f.Users[id]; ok {
		return u, nil
	}
	return repositories.User{}, pgx.ErrNoRows
}

func (f *Store) CreateSession(_ context.Context, arg repositories.CreateSessionParams) error {
	f.Calls["CreateSession"]++
	f.Sessions[string(arg.TokenHash)] = repositories.Session{
		TokenHash: arg.TokenHash, UserID: arg.UserID, CreatedAt: f.Now, ExpiresAt: arg.ExpiresAt,
	}
	return nil
}

func (f *Store) GetSessionUserID(_ context.Context, tokenHash []byte) (int64, error) {
	s, ok := f.Sessions[string(tokenHash)]
	if !ok || !s.ExpiresAt.After(f.Now) {
		return 0, pgx.ErrNoRows
	}
	return s.UserID, nil
}

func (f *Store) DeleteSession(_ context.Context, tokenHash []byte) error {
	delete(f.Sessions, string(tokenHash))
	return nil
}

func (f *Store) DeleteExpiredSessions(context.Context) (int64, error) {
	var n int64
	for k, s := range f.Sessions {
		if !s.ExpiresAt.After(f.Now) {
			delete(f.Sessions, k)
			n++
		}
	}
	return n, nil
}

func (f *Store) RegisterLoginAttempt(_ context.Context, email string) (repositories.RegisterLoginAttemptRow, error) {
	f.Calls["RegisterLoginAttempt"]++
	f.Attempts[email]++
	return repositories.RegisterLoginAttemptRow{Attempts: f.Attempts[email], WindowEnd: f.WindowEnd}, nil
}

func (f *Store) ClearLoginAttempts(_ context.Context, email string) error {
	f.Calls["ClearLoginAttempts"]++
	delete(f.Attempts, email)
	return nil
}

func (f *Store) DeleteStaleLoginAttempts(context.Context) (int64, error) {
	return 0, nil
}

func (f *Store) CreateTask(_ context.Context, arg repositories.CreateTaskParams) (repositories.Task, error) {
	t := repositories.Task{ID: f.id(), UserID: arg.UserID, Title: arg.Title, CreatedAt: f.Now, UpdatedAt: f.Now}
	f.Tasks[t.ID] = t
	return t, nil
}

func (f *Store) ListTasks(_ context.Context, arg repositories.ListTasksParams) ([]repositories.Task, error) {
	out := []repositories.Task{}
	for _, t := range f.Tasks {
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

func (f *Store) UpdateTask(_ context.Context, arg repositories.UpdateTaskParams) (repositories.Task, error) {
	t, ok := f.Tasks[arg.ID]
	if !ok || t.UserID != arg.UserID {
		return repositories.Task{}, pgx.ErrNoRows
	}
	if arg.Title.Valid {
		t.Title = arg.Title.String
	}
	if arg.Completed.Valid {
		t.Completed = arg.Completed.Bool
	}
	t.UpdatedAt = f.Now
	f.Tasks[t.ID] = t
	return t, nil
}

func (f *Store) DeleteTask(_ context.Context, arg repositories.DeleteTaskParams) (int64, error) {
	t, ok := f.Tasks[arg.ID]
	if !ok || t.UserID != arg.UserID {
		return 0, nil
	}
	delete(f.Tasks, arg.ID)
	return 1, nil
}

// Passwords stores "hash:" + password and counts calls, so tests can tell
// whether a password was checked at all.
type Passwords struct {
	Hashes, Verifies, Dummies int
}

func (p *Passwords) Hash(password string) string {
	p.Hashes++
	return "hash:" + password
}

func (p *Passwords) Verify(password, encoded string) (bool, error) {
	p.Verifies++
	return encoded == "hash:"+password, nil
}

func (p *Passwords) VerifyDummy(string) {
	p.Dummies++
}
