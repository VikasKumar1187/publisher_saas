// Package checkgrp maintains the group of handlers for health checking.
package checkgrp

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	database "github.com/vikaskumar1187/publisher_saas/services/publisher/internal/sys/database/pgx"
	"github.com/vikaskumar1187/publisher_saas/services/publisher/pkg/web"
)

// Handlers manages the set of check endpoints.
type Handlers struct {
	build string
	db    *sqlx.DB
}

// New constructs a Handlers api for the check group.
func New(build string, db *sqlx.DB) *Handlers {
	return &Handlers{
		build: build,
		db:    db,
	}
}

// Test is our example route.
func (h Handlers) Test(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	status := "ok"
	statusCode := http.StatusOK

	data := struct {
		Status string `json:"status"`
	}{
		Status: status,
	}

	return web.Respond(ctx, w, data, statusCode)
}

// Readiness checks if the database is ready and if not will return a 500 status.
// Do not respond by just returning an error because further up in the call
// stack it will interpret that as a non-trusted error.
func (h Handlers) Readiness(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()

	status := "ok"
	statusCode := http.StatusOK
	if err := database.StatusCheck(ctx, h.db); err != nil {
		status = "db not ready"
		statusCode = http.StatusInternalServerError
	}

	data := struct {
		Status string `json:"status"`
	}{
		Status: status,
	}

	return web.Respond(ctx, w, data, statusCode)
}

// Liveness returns simple status info if the service is alive. If the
// app is deployed to a Kubernetes cluster, it will also return pod, node, and
// namespace details via the Downward API. The Kubernetes environment variables
// need to be set within your Pod/Deployment manifest.
func (h Handlers) Liveness(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	host, err := os.Hostname()
	if err != nil {
		host = "unavailable"
	}

	data := struct {
		Status    string `json:"status,omitempty"`
		Build     string `json:"build,omitempty"`
		Host      string `json:"host,omitempty"`
		Name      string `json:"name,omitempty"`
		PodIP     string `json:"podIP,omitempty"`
		Node      string `json:"node,omitempty"`
		Namespace string `json:"namespace,omitempty"`
	}{
		Status:    "up",
		Build:     h.build,
		Host:      host,
		Name:      os.Getenv("KUBERNETES_NAME"),
		PodIP:     os.Getenv("KUBERNETES_POD_IP"),
		Node:      os.Getenv("KUBERNETES_NODE_NAME"),
		Namespace: os.Getenv("KUBERNETES_NAMESPACE"),
	}

	statusCode := http.StatusOK
	return web.Respond(ctx, w, data, statusCode)

}
