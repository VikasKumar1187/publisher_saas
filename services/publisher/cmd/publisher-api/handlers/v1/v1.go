// Package v1 contains the full set of handler functions and routes
// supported by the v1 web api.
package v1

import (
	"github.com/jmoiron/sqlx"
	"github.com/vikaskumar1187/publisher_saas/services/publisher/internal/web/auth"
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

}
