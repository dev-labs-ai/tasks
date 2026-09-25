package services

import (
	"errors"
	"testing"
	"time"

	"github.com/dev-labs-ai/tasks/apps/api/internal/auth"
)

var testNow = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)

func newAuthService() (*AuthService, *fakeStore, *fakePasswords) {
	store := newFakeStore(testNow)
	passwords := &fakePasswords{}
	s := NewAuthService(store, passwords)
	s.now = func() time.Time { return testNow }
	return s, store, passwords
}

func TestRegisterCreatesUserAndSession(t *testing.T) {
	s, store, _ := newAuthService()

	session, err := s.Register(t.Context(), "  Ana@Example.com ", "correct horse")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if session.User.Email != "ana@example.com" {
		t.Errorf("email = %q, want it trimmed and lowercased", session.User.Email)
	}
	if want := testNow.Add(24 * time.Hour); !session.ExpiresAt.Equal(want) {
		t.Errorf("expires at %v, want %v", session.ExpiresAt, want)
	}
	stored, ok := store.sessions[string(auth.HashSessionToken(session.Token))]
	if !ok || stored.UserID != session.User.ID {
		t.Errorf("no session stored under the token's hash for user %d", session.User.ID)
	}
	if store.users[session.User.ID].PasswordHash != "hash:correct horse" {
		t.Error("the stored password is not the hash")
	}
	if store.calls["InTx"] != 1 {
		t.Errorf("InTx called %d times, want user and session in one transaction", store.calls["InTx"])
	}
}

func TestRegisterValidatesEveryField(t *testing.T) {
	s, store, passwords := newAuthService()

	_, err := s.Register(t.Context(), "Ana <ana@example.com>", "short")

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want *ValidationError", err)
	}
	if ve.Fields["email"] == "" || ve.Fields["password"] == "" {
		t.Errorf("fields = %v, want problems for email and password", ve.Fields)
	}
	if passwords.hashes != 0 || store.calls["CreateUser"] != 0 {
		t.Error("invalid input reached hashing or the database")
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	s, _, _ := newAuthService()
	if _, err := s.Register(t.Context(), "ana@example.com", "correct horse"); err != nil {
		t.Fatalf("first Register: %v", err)
	}

	_, err := s.Register(t.Context(), "ANA@example.com", "another password")
	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("err = %v, want ErrEmailTaken", err)
	}
}

func TestLoginFailuresLookTheSame(t *testing.T) {
	s, _, passwords := newAuthService()
	if _, err := s.Register(t.Context(), "ana@example.com", "correct horse"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, unknownErr := s.Login(t.Context(), "nobody@example.com", "correct horse")
	_, wrongErr := s.Login(t.Context(), "ana@example.com", "wrong password")

	if !errors.Is(unknownErr, ErrInvalidCredentials) || !errors.Is(wrongErr, ErrInvalidCredentials) {
		t.Errorf("unknown email: %v; wrong password: %v; want ErrInvalidCredentials for both", unknownErr, wrongErr)
	}
	if passwords.dummies != 1 {
		t.Errorf("VerifyDummy called %d times, want once for the unknown email", passwords.dummies)
	}
}

func TestLoginRefusesTheSixthAttemptWithoutCheckingThePassword(t *testing.T) {
	s, store, passwords := newAuthService()
	if _, err := s.Register(t.Context(), "ana@example.com", "correct horse"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	for i := 1; i <= MaxLoginAttempts; i++ {
		if _, err := s.Login(t.Context(), "ana@example.com", "wrong password"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d: err = %v, want ErrInvalidCredentials", i, err)
		}
	}
	checked := passwords.verifies

	_, err := s.Login(t.Context(), "ana@example.com", "correct horse")

	var tooMany *TooManyAttemptsError
	if !errors.As(err, &tooMany) {
		t.Fatalf("6th attempt: err = %v, want *TooManyAttemptsError", err)
	}
	if !tooMany.RetryAt.Equal(windowEnd) {
		t.Errorf("RetryAt = %v, want the window end %v", tooMany.RetryAt, windowEnd)
	}
	if passwords.verifies != checked || passwords.dummies != 0 {
		t.Error("the refused attempt still checked the password")
	}
	if store.calls["CreateSession"] != 1 {
		t.Error("the refused attempt created a session")
	}
}

func TestLoginUnknownEmailsAreLimitedToo(t *testing.T) {
	s, _, _ := newAuthService()

	for range MaxLoginAttempts {
		_, _ = s.Login(t.Context(), "nobody@example.com", "guess")
	}
	_, err := s.Login(t.Context(), "nobody@example.com", "guess")

	var tooMany *TooManyAttemptsError
	if !errors.As(err, &tooMany) {
		t.Errorf("err = %v, want *TooManyAttemptsError: otherwise the limit reveals which emails exist", err)
	}
}

func TestLoginSuccessClearsAttemptsAndOpensSession(t *testing.T) {
	s, store, _ := newAuthService()
	if _, err := s.Register(t.Context(), "ana@example.com", "correct horse"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	_, _ = s.Login(t.Context(), "ana@example.com", "wrong password")

	session, err := s.Login(t.Context(), "Ana@example.com", "correct horse")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	if _, counted := store.attempts["ana@example.com"]; counted || store.calls["ClearLoginAttempts"] != 1 {
		t.Error("a successful login did not clear the attempts")
	}
	if userID, err := s.Authenticate(t.Context(), session.Token); err != nil || userID != session.User.ID {
		t.Errorf("Authenticate(new token) = %d, %v; want user %d", userID, err, session.User.ID)
	}
}

func TestLoginValidatesInputBeforeCounting(t *testing.T) {
	s, store, _ := newAuthService()

	_, err := s.Login(t.Context(), "not an email", "")

	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Fields["email"] == "" || ve.Fields["password"] == "" {
		t.Errorf("err = %v, want a *ValidationError for email and password", err)
	}
	if store.calls["RegisterLoginAttempt"] != 0 {
		t.Error("a malformed request touched the attempt counter")
	}
}

func TestAuthenticateAndLogout(t *testing.T) {
	s, store, _ := newAuthService()
	session, err := s.Register(t.Context(), "ana@example.com", "correct horse")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	for _, token := range []string{"", "not-a-real-token"} {
		if _, err := s.Authenticate(t.Context(), token); !errors.Is(err, ErrUnauthenticated) {
			t.Errorf("Authenticate(%q): err = %v, want ErrUnauthenticated", token, err)
		}
	}

	store.now = testNow.Add(SessionTTL)
	if _, err := s.Authenticate(t.Context(), session.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("expired session: err = %v, want ErrUnauthenticated", err)
	}
	store.now = testNow

	if err := s.Logout(t.Context(), session.Token); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := s.Authenticate(t.Context(), session.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("after logout: err = %v, want ErrUnauthenticated", err)
	}
	if err := s.Logout(t.Context(), ""); err != nil {
		t.Errorf("Logout without a token: %v", err)
	}
}

func TestUser(t *testing.T) {
	s, _, _ := newAuthService()
	session, err := s.Register(t.Context(), "ana@example.com", "correct horse")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	user, err := s.User(t.Context(), session.User.ID)
	if err != nil || user.Email != "ana@example.com" {
		t.Errorf("User = %+v, %v; want ana", user, err)
	}
	if _, err := s.User(t.Context(), 999); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("unknown user: err = %v, want ErrUnauthenticated", err)
	}
}
