// Package handlers manages the different versions of the API.
package handlers

import (
	"context"
	"net/http"
	"os"

	"github.com/jmoiron/sqlx"
	v1 "github.com/vikaskumar1187/publisher_saas/services/publisher/cmd/publisher-api/handlers/v1"
	"github.com/vikaskumar1187/publisher_saas/services/publisher/internal/web/auth"
	"github.com/vikaskumar1187/publisher_saas/services/publisher/internal/web/v1/mid"
	"github.com/vikaskumar1187/publisher_saas/services/publisher/pkg/web"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Options represent optional parameters.
type Options struct {
	corsOrigin string
}

// WithCORS provides configuration options for CORS.
func WithCORS(origin string) func(opts *Options) {
	return func(opts *Options) {
		opts.corsOrigin = origin
	}
}

// APIMuxConfig contains all the mandatory systems required by handlers.
type APIMuxConfig struct {
	Build    string
	Shutdown chan os.Signal
	Log      *zap.SugaredLogger
	Auth     *auth.Auth
	DB       *sqlx.DB
	Tracer   trace.Tracer
}

// APIMux constructs an http.Handler with all application routes defined.
// It takes a configuration (APIMuxConfig) and optional functional options.
//
// Parameters:
// - cfg: APIMuxConfig containing all mandatory systems required by handlers.
// - options: Variadic functional options for additional configurations.
//
// Returns:
// - http.Handler: The constructed handler with all routes defined.
func APIMux(cfg APIMuxConfig, options ...func(opts *Options)) http.Handler {

	var opts Options
	for _, option := range options {
		option(&opts)
	}

	var app *web.App

	if opts.corsOrigin != "" {
		app = web.NewApp(
			cfg.Shutdown,
			cfg.Tracer,
			mid.Logger(cfg.Log),
			mid.Errors(cfg.Log),
			mid.Metrics(),
			mid.Cors(opts.corsOrigin),
			mid.Panics(),
		)

		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			return nil
		}
		app.Handle(http.MethodOptions, "", "/*", h, mid.Cors(opts.corsOrigin))
	}

	if app == nil {
		app = web.NewApp(
			cfg.Shutdown,
			cfg.Tracer,
			mid.Logger(cfg.Log),
			mid.Errors(cfg.Log),
			mid.Metrics(),
			mid.Panics(),
			mid.ResponseWriter(cfg.Log),
		)
	}

	// Define the routes for the application
	// This is where we set up all the handlers for different API endpoints
	// The v1.Routes function is called to set up the version 1 API routes
	// It takes the web.App instance and a configuration struct as parameters

	v1.Routes(app, v1.Config{
		Build: cfg.Build,
		Log:   cfg.Log,
		Auth:  cfg.Auth,
		DB:    cfg.DB,
	})

	return app
}
