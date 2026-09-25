package handlers

import "net/http"

// Deps are what the routes need.
type Deps struct {
	DB   Pinger
	Auth AuthService
	// SecureCookies marks the session cookie Secure; true in production.
	SecureCookies bool
}

// NewRouter registers the API's routes (tech spec §6.2). Anything else,
// including a known path with an unsupported method, answers NOT_FOUND.
func NewRouter(d Deps) *http.ServeMux {
	mux := http.NewServeMux()
	authed := requireAuth(d.Auth)
	a := &authHandlers{svc: d.Auth, secureCookies: d.SecureCookies}

	mux.Handle("GET /healthz", Health(d.DB))

	mux.HandleFunc("POST /api/v1/auth/register", a.register)
	mux.HandleFunc("POST /api/v1/auth/login", a.login)
	mux.Handle("POST /api/v1/auth/logout", authed(http.HandlerFunc(a.logout)))
	mux.Handle("GET /api/v1/auth/me", authed(http.HandlerFunc(a.me)))

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		WriteError(w, NotFound())
	})
	return mux
}
