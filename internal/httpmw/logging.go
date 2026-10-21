package httpmw

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// statusRecorder wraps a ResponseWriter to capture the status code that was
// actually sent, defaulting to 200 if the handler never calls WriteHeader
// explicitly (http.ResponseWriter does the same on the first Write). It
// forwards the optional Flusher/Hijacker interfaces so wrapping doesn't
// silently break handlers that need to stream or take over the connection.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("httpmw: underlying ResponseWriter does not support hijacking")
	}
	return h.Hijack()
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
