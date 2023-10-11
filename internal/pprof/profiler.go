package pprof

import (
	"net/http"
	"net/http/pprof"
	"strings"
)

const DefaultPrefix = "/debug/pprof/"

var handler http.Handler

func init() {
	mux := http.NewServeMux()

	mux.HandleFunc(DefaultPrefix, pprof.Index)
	mux.HandleFunc(DefaultPrefix+"cmdline", pprof.Cmdline)
	mux.HandleFunc(DefaultPrefix+"profile", pprof.Profile)
	mux.HandleFunc(DefaultPrefix+"symbol", pprof.Symbol)
	mux.HandleFunc(DefaultPrefix+"trace", pprof.Trace)

	handler = mux
}

// Handler returns a pprof handler for the routes: /debug/pprof/*
func Handler() http.Handler {
	return handler
}

// Middleware execute pprof handler if the request path starts with /debug/pprof/ or the next handler otherwise
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, DefaultPrefix) {
			handler.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
