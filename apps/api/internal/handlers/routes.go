package handlers

import "net/http"

// NewRouter registers the API's routes. Anything else, including a known
// path with an unsupported method, answers NOT_FOUND.
func NewRouter(db Pinger) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", Health(db))
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		WriteError(w, NotFound())
	})
	return mux
}
