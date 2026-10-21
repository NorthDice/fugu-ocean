// Package httpmw holds cross-cutting HTTP middleware (request logging,
// panic recovery, ...) and the helper that chains them around a handler.
package httpmw

import "net/http"

// Middleware wraps an http.Handler with additional behavior.
type Middleware func(http.Handler) http.Handler

// Chain wraps h with mw in order: the first middleware listed is the
// outermost one, so it sees a request before any of the others do.
func Chain(h http.Handler, mw ...Middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
