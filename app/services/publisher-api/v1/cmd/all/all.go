// Package all binds all the routes into the specified app.
package all

import (
	"github.com/vikaskumar1187/publisher_saas/app/services/publisher-api/v1/handlers/checkgrp"
	v1 "github.com/vikaskumar1187/publisher_saas/business/web/v1"
	"github.com/vikaskumar1187/publisher_saas/foundation/web"
)

// Routes constructs the add value which provides the implementation of
// of RouteAdder for specifying what routes to bind to this instance.
func Routes() add {
	return add{}
}

type add struct{}

// Add implements the RouterAdder interface.
func (add) Add(app *web.App, cfg v1.APIMuxConfig) {
	checkgrp.Routes(app, checkgrp.Config{
		UsingWeaver: cfg.UsingWeaver,
		Build:       cfg.Build,
		Log:         cfg.Log,
		DB:          cfg.DB,
	})

}
