// Package v1 contains the full set of handler functions and routes
// supported by the v1 web api.
package v1

import (
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/vikaskumar1187/publisher_saas/services/publisher/internal/web/auth"
	"github.com/vikaskumar1187/publisher_saas/services/publisher/internal/web/v1/debug/checkgrp"
	"github.com/vikaskumar1187/publisher_saas/services/publisher/internal/web/v1/mid"
	"github.com/vikaskumar1187/publisher_saas/services/publisher/pkg/web"
	"go.uber.org/zap"
)

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Build string
	Log   *zap.SugaredLogger
	Auth  *auth.Auth
	DB    *sqlx.DB
}

// Routes binds all the version 1 routes.
func Routes(app *web.App, cfg Config) {
	const version = "v1"

	// -------------------------------------------------------------------------

	cgh := checkgrp.New(cfg.Build, cfg.DB)

	app.Handle(http.MethodGet, version, "/readiness", cgh.Readiness)
	app.Handle(http.MethodGet, version, "/liveness", cgh.Liveness)
	app.Handle(http.MethodGet, version, "/test", cgh.Test)
	app.Handle(http.MethodGet, version, "/test-auth", cgh.TestAuth, mid.Authenticate(cfg.Auth))

	// Add this catch-all route at the end
	app.Handle("*", version, "/*", cgh.HandleNotFound)
}
