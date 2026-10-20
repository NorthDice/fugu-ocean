package httpmw

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/NorthDice/fugu-ocean/internal/httpapi"
)

// Recover turns a panic anywhere downstream into a logged stack trace and a
// JSON 500 response, instead of crashing the server process.
func Recover(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Printf("panic recovered: %v\n%s", rec, debug.Stack())
					httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
