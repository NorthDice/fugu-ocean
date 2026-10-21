package httpmw

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime/debug"

	"github.com/NorthDice/fugu-ocean/internal/httpapi"
)

// Recover turns a panic anywhere downstream into a logged stack trace and a
// JSON 500 response, instead of crashing the server process. If the handler
// had already started writing its response before panicking, writing the
// JSON error would only corrupt that response, so Recover logs the panic and
// leaves the response as-is in that case.
func Recover(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wt := &writeTracker{ResponseWriter: w}
			defer func() {
				if rec := recover(); rec != nil {
					logger.Printf("panic recovered: %v\n%s", rec, debug.Stack())
					if wt.wrote {
						logger.Print("response already started; cannot send error response")
						return
					}
					httpapi.WriteError(wt, http.StatusInternalServerError, "internal_error", "internal server error")
				}
			}()
			next.ServeHTTP(wt, r)
		})
	}
}

// writeTracker wraps a ResponseWriter to record whether a response has
// already been started, and forwards the optional Flusher/Hijacker
// interfaces so wrapping doesn't silently break handlers that need them.
type writeTracker struct {
	http.ResponseWriter
	wrote bool
}

func (w *writeTracker) WriteHeader(status int) {
	w.wrote = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *writeTracker) Write(b []byte) (int, error) {
	w.wrote = true
	return w.ResponseWriter.Write(b)
}

func (w *writeTracker) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *writeTracker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("httpmw: underlying ResponseWriter does not support hijacking")
	}
	w.wrote = true
	return h.Hijack()
}
