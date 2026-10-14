// Package middleware holds HTTP middleware shared by all routes.
package middleware

import (
	"log"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// LogRecover logs each request and turns handler panics into a 500 response.
func LogRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			if err := recover(); err != nil {
				if err == http.ErrAbortHandler {
					panic(err)
				}
				log.Printf("panic: %v", err)
				rec.WriteHeader(http.StatusInternalServerError)
			}
			log.Printf("%s %s from %s - %d in %v", r.Method, r.URL, r.RemoteAddr, rec.status, time.Since(start))
		}()
		next.ServeHTTP(rec, r)
	})
}
