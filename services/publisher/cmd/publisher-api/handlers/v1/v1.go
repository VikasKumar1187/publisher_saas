// Package v1 contains the full set of handler functions and routes
// supported by the v1 web api.
package v1

import (
	"net/http"

	"github.com/ardanlabs/service/app/sdk/auth"
	"github.com/ardanlabs/service/business/domain/productbus/stores/productdb"
	"github.com/ardanlabs/service/business/domain/userbus/stores/usercache"
	"github.com/ardanlabs/service/business/domain/userbus/stores/userdb"
	"github.com/jmoiron/sqlx"
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

	envCore := event.NewCore(cfg.Log)
	usrCore := user.NewCore(envCore, usercache.NewStore(cfg.Log, userdb.NewStore(cfg.Log, cfg.DB)))
	prdCore := product.NewCore(cfg.Log, envCore, usrCore, productdb.NewStore(cfg.Log, cfg.DB))
	smmCore := summary.NewCore(summarydb.NewStore(cfg.Log, cfg.DB))

	authen := mid.Authenticate(cfg.Auth)
	ruleAdmin := mid.Authorize(cfg.Auth, auth.RuleAdminOnly)
	ruleAdminOrSubject := mid.Authorize(cfg.Auth, auth.RuleAdminOrSubject)

	// -------------------------------------------------------------------------

	cgh := checkgrp.New(cfg.Build, cfg.DB)

	app.Handle(http.MethodGet, version, "/readiness", cgh.Readiness)
	app.Handle(http.MethodGet, version, "/liveness", cgh.Liveness)

	// -------------------------------------------------------------------------

	ugh := usergrp.New(usrCore, smmCore, cfg.Auth)

	app.Handle(http.MethodGet, version, "/users/token/:kid", ugh.Token)
	app.Handle(http.MethodGet, version, "/users", ugh.Query, authen, ruleAdmin)
	app.Handle(http.MethodGet, version, "/users/:user_id", ugh.QueryByID, authen, ruleAdminOrSubject)
	app.Handle(http.MethodGet, version, "/users/summary", ugh.QuerySummary, authen, ruleAdmin)
	app.Handle(http.MethodPost, version, "/users", ugh.Create, authen, ruleAdmin)
	app.Handle(http.MethodPut, version, "/users/:user_id", ugh.Update, authen, ruleAdminOrSubject)
	app.Handle(http.MethodDelete, version, "/users/:user_id", ugh.Delete, authen, ruleAdminOrSubject)

	// -------------------------------------------------------------------------

	pgh := productgrp.New(prdCore, usrCore, cfg.Auth)

	app.Handle(http.MethodGet, version, "/products", pgh.Query, authen)
	app.Handle(http.MethodGet, version, "/products/:product_id", pgh.QueryByID, authen)
	app.Handle(http.MethodPost, version, "/products", pgh.Create, authen)
	app.Handle(http.MethodPut, version, "/products/:product_id", pgh.Update, authen)
	app.Handle(http.MethodDelete, version, "/products/:product_id", pgh.Delete, authen)
}
