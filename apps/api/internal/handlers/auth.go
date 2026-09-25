package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/dev-labs-ai/tasks/apps/api/internal/services"
)

// SessionCookie is the name of the cookie that carries the session token.
const SessionCookie = "session"

// AuthService is the part of services.AuthService the handlers use.
type AuthService interface {
	Register(ctx context.Context, email, password string) (services.Session, error)
	Login(ctx context.Context, email, password string) (services.Session, error)
	Logout(ctx context.Context, token string) error
	Authenticate(ctx context.Context, token string) (int64, error)
	User(ctx context.Context, id int64) (services.User, error)
}

type userResponse struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

// newUserResponse lists the fields sent to clients one by one, so a field
// added to the user later, like a password hash, never leaks by default.
func newUserResponse(u services.User) userResponse {
	return userResponse{ID: u.ID, Email: u.Email, CreatedAt: u.CreatedAt.UTC()}
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authHandlers struct {
	svc AuthService
	// secureCookies adds the Secure attribute; production only, because
	// development runs on plain HTTP.
	secureCookies bool
}

func (h *authHandlers) register(w http.ResponseWriter, r *http.Request) {
	var req credentials
	if e := DecodeJSON(w, r, &req); e != nil {
		WriteError(w, e)
		return
	}
	session, err := h.svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	h.setSessionCookie(w, session.Token)
	WriteData(w, http.StatusCreated, newUserResponse(session.User))
}

func (h *authHandlers) login(w http.ResponseWriter, r *http.Request) {
	var req credentials
	if e := DecodeJSON(w, r, &req); e != nil {
		WriteError(w, e)
		return
	}
	session, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	h.setSessionCookie(w, session.Token)
	WriteData(w, http.StatusOK, newUserResponse(session.User))
}

func (h *authHandlers) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Logout(r.Context(), sessionToken(r)); err != nil {
		writeServiceError(w, r, err)
		return
	}
	http.SetCookie(w, h.sessionCookie("", -1))
	w.WriteHeader(http.StatusNoContent)
}

func (h *authHandlers) me(w http.ResponseWriter, r *http.Request) {
	user, err := h.svc.User(r.Context(), userID(r.Context()))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	WriteData(w, http.StatusOK, newUserResponse(user))
}

func (h *authHandlers) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, h.sessionCookie(token, int(services.SessionTTL.Seconds())))
}

// sessionCookie builds the cookie from tech spec §5.5. A negative maxAge
// deletes it.
func (h *authHandlers) sessionCookie(token string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	}
}

func sessionToken(r *http.Request) string {
	c, err := r.Cookie(SessionCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

type userIDKey struct{}

// requireAuth lets a request through only with a valid session cookie,
// and puts the session's user ID in its context.
func requireAuth(svc AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := svc.Authenticate(r.Context(), sessionToken(r))
			if err != nil {
				writeServiceError(w, r, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey{}, id)))
		})
	}
}

// userID returns the user that requireAuth authenticated. It panics on a
// route registered without requireAuth, which is a programming error.
func userID(ctx context.Context) int64 {
	id, ok := ctx.Value(userIDKey{}).(int64)
	if !ok {
		panic("handlers: route is missing requireAuth")
	}
	return id
}
