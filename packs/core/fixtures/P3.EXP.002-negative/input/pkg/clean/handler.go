package clean

import "net/http"

// RegisterRoutes explicitly registers routes — no init() magic.
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", healthHandler)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}
