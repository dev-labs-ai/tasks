// Package services holds the business rules for authentication and tasks.
package services

import (
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"unicode/utf8"
)

// Limits from tech spec §5.6. Lengths count Unicode code points, as
// PostgreSQL's char_length does.
const (
	MaxEmailLength    = 254
	MinPasswordLength = 8
	MaxPasswordLength = 64
	MaxTitleLength    = 200
)

// ValidationError lists what is wrong with each field of a request. The
// handlers turn it into a VALIDATION_FAILED response with these fields.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	names := make([]string, 0, len(e.Fields))
	for name := range e.Fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return "invalid fields: " + strings.Join(names, ", ")
}

// fieldErrors collects problems and becomes a *ValidationError only when
// there is at least one.
type fieldErrors map[string]string

func (f fieldErrors) add(field, problem string) {
	if problem != "" {
		f[field] = problem
	}
}

func (f fieldErrors) err() error {
	if len(f) == 0 {
		return nil
	}
	return &ValidationError{Fields: f}
}

// ValidateEmail trims and lowercases raw and checks it is a bare address:
// "Ana <ana@example.com>" is rejected. It returns the normalized email, or
// a problem to show the user.
func ValidateEmail(raw string) (email, problem string) {
	email = strings.ToLower(strings.TrimSpace(raw))
	switch {
	case email == "":
		return "", "Email is required"
	case utf8.RuneCountInString(email) > MaxEmailLength:
		return "", fmt.Sprintf("Email must have at most %d characters", MaxEmailLength)
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", "Enter a valid email address"
	}
	return email, ""
}

// ValidatePassword checks the length of password. Passwords are never
// trimmed: spaces are part of the password.
func ValidatePassword(password string) (problem string) {
	if n := utf8.RuneCountInString(password); n < MinPasswordLength || n > MaxPasswordLength {
		return fmt.Sprintf("Password must have %d to %d characters", MinPasswordLength, MaxPasswordLength)
	}
	return ""
}

// ValidateTitle trims raw and checks its length. It returns the trimmed
// title, or a problem to show the user.
func ValidateTitle(raw string) (title, problem string) {
	title = strings.TrimSpace(raw)
	switch n := utf8.RuneCountInString(title); {
	case n == 0:
		return "", "Title is required"
	case n > MaxTitleLength:
		return "", fmt.Sprintf("Title must have at most %d characters", MaxTitleLength)
	}
	return title, ""
}

// TaskStatus filters the task list.
type TaskStatus int

const (
	StatusAll TaskStatus = iota
	StatusPending
	StatusCompleted
)

// ParseStatus reads the status query parameter. An absent value means all
// tasks.
func ParseStatus(raw string) (status TaskStatus, problem string) {
	switch raw {
	case "":
		return StatusAll, ""
	case "pending":
		return StatusPending, ""
	case "completed":
		return StatusCompleted, ""
	}
	return StatusAll, "Status must be pending or completed"
}
