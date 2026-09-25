package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dev-labs-ai/tasks/apps/api/internal/auth"
	"github.com/dev-labs-ai/tasks/apps/api/internal/repositories"
)

const (
	// SessionTTL is how long a session lasts after login, with no renewal
	// (PRD FR-2).
	SessionTTL = 24 * time.Hour

	// MaxLoginAttempts is how many logins an email gets per window. The
	// window length lives in queries/login_attempts.sql.
	MaxLoginAttempts = 5
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthenticated    = errors.New("no valid session")
)

// TooManyAttemptsError means the email spent its login attempts for the
// current window.
type TooManyAttemptsError struct {
	RetryAt time.Time
}

func (e *TooManyAttemptsError) Error() string {
	return "too many login attempts; retry at " + e.RetryAt.Format(time.RFC3339)
}

// Passwords hashes and verifies passwords; auth.Passwords is the real one.
type Passwords interface {
	Hash(password string) string
	Verify(password, encoded string) (bool, error)
	// VerifyDummy costs as much as Verify and discards the result.
	VerifyDummy(password string)
}

type User struct {
	ID        int64
	Email     string
	CreatedAt time.Time
}

// Session is a new login: the token goes into the cookie.
type Session struct {
	Token     string
	ExpiresAt time.Time
	User      User
}

type AuthService struct {
	store     Store
	passwords Passwords
	now       func() time.Time
}

func NewAuthService(store Store, passwords Passwords) *AuthService {
	return &AuthService{store: store, passwords: passwords, now: time.Now}
}

// Register creates the account and its first session in one transaction,
// so the user is logged in as soon as the account exists (PRD FR-1).
func (s *AuthService) Register(ctx context.Context, email, password string) (Session, error) {
	fields := fieldErrors{}
	email, problem := ValidateEmail(email)
	fields.add("email", problem)
	fields.add("password", ValidatePassword(password))
	if err := fields.err(); err != nil {
		return Session{}, err
	}

	// Hash before the transaction: it is the slow part.
	hash := s.passwords.Hash(password)

	var session Session
	err := s.store.InTx(ctx, func(q repositories.Querier) error {
		user, err := q.CreateUser(ctx, repositories.CreateUserParams{Email: email, PasswordHash: hash})
		if isUniqueViolation(err) {
			return ErrEmailTaken
		}
		if err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		session, err = s.createSession(ctx, q, user)
		return err
	})
	return session, err
}

// Login checks the credentials and opens a new session. The attempt is
// counted before the password is checked, so parallel requests cannot get
// past the limit (tech spec §5.5).
func (s *AuthService) Login(ctx context.Context, email, password string) (Session, error) {
	fields := fieldErrors{}
	email, problem := ValidateEmail(email)
	fields.add("email", problem)
	if password == "" {
		fields.add("password", "Password is required")
	}
	if err := fields.err(); err != nil {
		return Session{}, err
	}

	attempt, err := s.store.RegisterLoginAttempt(ctx, email)
	if err != nil {
		return Session{}, fmt.Errorf("register login attempt: %w", err)
	}
	if attempt.Attempts > MaxLoginAttempts {
		return Session{}, &TooManyAttemptsError{RetryAt: attempt.WindowEnd}
	}

	user, err := s.store.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		s.passwords.VerifyDummy(password)
		return Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, fmt.Errorf("get user: %w", err)
	}

	ok, err := s.passwords.Verify(password, user.PasswordHash)
	if err != nil {
		return Session{}, fmt.Errorf("verify password of user %d: %w", user.ID, err)
	}
	if !ok {
		return Session{}, ErrInvalidCredentials
	}

	if err := s.store.ClearLoginAttempts(ctx, email); err != nil {
		return Session{}, fmt.Errorf("clear login attempts: %w", err)
	}
	return s.createSession(ctx, s.store, user)
}

// Logout ends the session. An unknown or empty token is not an error.
func (s *AuthService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	if err := s.store.DeleteSession(ctx, auth.HashSessionToken(token)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// Authenticate returns the ID of the user who owns an unexpired session.
func (s *AuthService) Authenticate(ctx context.Context, token string) (int64, error) {
	if token == "" {
		return 0, ErrUnauthenticated
	}
	userID, err := s.store.GetSessionUserID(ctx, auth.HashSessionToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrUnauthenticated
	}
	if err != nil {
		return 0, fmt.Errorf("get session: %w", err)
	}
	return userID, nil
}

// User returns the account behind an authenticated request.
func (s *AuthService) User(ctx context.Context, id int64) (User, error) {
	user, err := s.store.GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUnauthenticated
	}
	if err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}
	return toUser(user), nil
}

func (s *AuthService) createSession(ctx context.Context, q repositories.Querier, user repositories.User) (Session, error) {
	token, hash := auth.NewSessionToken()
	expiresAt := s.now().Add(SessionTTL)

	err := q.CreateSession(ctx, repositories.CreateSessionParams{TokenHash: hash, UserID: user.ID, ExpiresAt: expiresAt})
	if err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}
	return Session{Token: token, ExpiresAt: expiresAt, User: toUser(user)}, nil
}

func toUser(u repositories.User) User {
	return User{ID: u.ID, Email: u.Email, CreatedAt: u.CreatedAt}
}
