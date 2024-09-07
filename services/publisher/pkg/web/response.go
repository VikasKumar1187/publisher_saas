package web

import (
	"context"
	"encoding/json"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
)

// StatusRecorder wraps http.ResponseWriter to capture the status code and header writes
type StatusRecorder struct {
	http.ResponseWriter
	Status       int
	HeaderWrites int
}

func (r *StatusRecorder) WriteHeader(statusCode int) {
	if r.Status != 0 {
		return // Ignore subsequent calls to WriteHeader
	}
	r.Status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *StatusRecorder) Header() http.Header {
	r.HeaderWrites++
	return r.ResponseWriter.Header()
}

// Respond converts a Go value to JSON and sends it to the client.
func Respond(ctx context.Context, w http.ResponseWriter, data any, statusCode int) error {
	ctx, span := AddSpan(ctx, "foundation.web.response", attribute.Int("status", statusCode))
	defer span.End()

	SetStatusCode(ctx, statusCode)

	recorder := &StatusRecorder{ResponseWriter: w, Status: 0}

	if statusCode == http.StatusNoContent {
		recorder.WriteHeader(statusCode)
		return nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	recorder.Header().Set("Content-Type", "application/json")
	recorder.WriteHeader(statusCode)

	if _, err := recorder.Write(jsonData); err != nil {
		return err
	}

	return nil
}
