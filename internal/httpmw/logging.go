package httpmw

import (
	"log"
	"net/http"
	"time"
)

// statusRecorder wraps a ResponseWriter to capture the status code that was
// actually sent, defaulting to 200 if the handler never calls WriteHeader
// explicitly (http.ResponseWriter does the same on the first Write).
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Logging logs one line per request: method, path, resulting status code,
// and how long the handler took.
func Logging(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			logger.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start))
		})
	}
}
