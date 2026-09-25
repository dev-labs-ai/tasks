package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

// Error codes returned by the API (tech spec §6.4).
const (
	CodeMalformedRequest   = "MALFORMED_REQUEST"
	CodeValidationFailed   = "VALIDATION_FAILED"
	CodeUnauthenticated    = "UNAUTHENTICATED"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeForbiddenOrigin    = "FORBIDDEN_ORIGIN"
	CodeTaskNotFound       = "TASK_NOT_FOUND"
	CodeNotFound           = "NOT_FOUND"
	CodeEmailTaken         = "EMAIL_TAKEN"
	CodePayloadTooLarge    = "PAYLOAD_TOO_LARGE"
	CodeTooManyAttempts    = "TOO_MANY_ATTEMPTS"
	CodeInternalError      = "INTERNAL_ERROR"
)

// MaxBodyBytes caps every request body.
const MaxBodyBytes = 64 << 10

// Error is an error response. It is written as {"error": {...}}.
type Error struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (e *Error) Error() string {
	return e.Code + ": " + e.Message
}

func MalformedRequest() *Error {
	return &Error{Status: http.StatusBadRequest, Code: CodeMalformedRequest, Message: "The request is malformed"}
}

func ValidationFailed(fields map[string]string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: CodeValidationFailed, Message: "The request has invalid fields", Fields: fields}
}

func NotFound() *Error {
	return &Error{Status: http.StatusNotFound, Code: CodeNotFound, Message: "Not found"}
}

func ForbiddenOrigin() *Error {
	return &Error{Status: http.StatusForbidden, Code: CodeForbiddenOrigin, Message: "Cross-origin requests are not allowed"}
}

func PayloadTooLarge() *Error {
	return &Error{Status: http.StatusRequestEntityTooLarge, Code: CodePayloadTooLarge, Message: "The request body must be at most 64 KiB"}
}

// InternalError carries no detail on purpose: the cause belongs in the
// logs, never in the response.
func InternalError() *Error {
	return &Error{Status: http.StatusInternalServerError, Code: CodeInternalError, Message: "Something went wrong"}
}

// DecodeJSON reads a JSON body into dst. Bodies over MaxBodyBytes, unknown
// fields, trailing data and anything that is not a single JSON value are
// rejected.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) *Error {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return decodeError(err)
	}
	return nil
}

func decodeError(err error) *Error {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return PayloadTooLarge()
	}
	return MalformedRequest()
}

// WriteData writes a success response as {"data": data}.
func WriteData(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, struct {
		Data any `json:"data"`
	}{data})
}

// WriteError writes an error response.
func WriteError(w http.ResponseWriter, e *Error) {
	writeJSON(w, e.Status, struct {
		Error *Error `json:"error"`
	}{e})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("write response", "error", err)
	}
}
