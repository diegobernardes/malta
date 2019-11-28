package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog"
)

// Logger used to record information about the requests.
func Logger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			t1 := time.Now()
			logger.Info().
				Str("endpoint", r.URL.String()).
				Str("method", r.Method).
				Msg("Request started")

			rw := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(rw, r)

			logger.Info().
				Dur("duration", time.Since(t1)).
				Int("status", rw.Status()).
				Msg("Request finished")
		}
		return http.HandlerFunc(fn)
	}
}
