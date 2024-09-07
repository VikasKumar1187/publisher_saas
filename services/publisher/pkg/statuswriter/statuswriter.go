package statuswriter

import (
	"net/http"

	"go.uber.org/zap"
)

// StatusRecorder wraps http.ResponseWriter to capture the status code and header writes
type StatusRecorder struct {
	http.ResponseWriter
	Status       int
	HeaderWrites int
	Log          *zap.SugaredLogger
}

// HeaderWrite represents a single header write operation
type HeaderWrite struct {
	Key   string
	Value []string
	File  string
	Line  int
}

// WriteHeader captures the status code and calls the underlying ResponseWriter's WriteHeader
func (r *StatusRecorder) WriteHeader(statusCode int) {
	if r.Status != 0 {
		r.Log.Infow("Superfluous WriteHeader call",
			"new_status", statusCode,
			"previous_status", r.Status,
		)
		return
	}
	r.Status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Header returns the header map that will be sent by WriteHeader
func (r *StatusRecorder) Header() http.Header {
	return r.ResponseWriter.Header()
}

// Write writes the data to the connection as part of an HTTP reply
func (r *StatusRecorder) Write(b []byte) (int, error) {
	if r.Status == 0 {
		r.Status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

// SetHeader captures header writes
func (r *StatusRecorder) SetHeader(key string, value []string) {
	r.HeaderWrites++
	r.Log.Infow("Header set",
		"key", key,
		"value", value,
	)
	for _, v := range value {
		r.ResponseWriter.Header().Add(key, v)
	}
}
