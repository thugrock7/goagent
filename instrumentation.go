package goagent

import (
	"net/http"
)

// Instrumentation defines the instrumentation elements for the APM.
// Every implementation is responsible to override them.
var Instrumentation = struct {
	// HttpHandler wraps a handler with instrumentation
	HTTPHandler func(http.Handler) http.Handler
}{
	// Default noop handler
	HTTPHandler: func(h http.Handler) http.Handler { return h },
}
