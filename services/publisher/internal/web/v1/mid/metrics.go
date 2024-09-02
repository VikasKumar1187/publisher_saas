package mid

import (
	"context"
	"net/http"

	"github.com/vikaskumar1187/publisher_saas/services/publisher/internal/web/metrics"
	"github.com/vikaskumar1187/publisher_saas/services/publisher/pkg/web"
)

// Metrics updates program counters.
func Metrics() web.Middleware {
	m := func(handler web.Handler) web.Handler {
		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			ctx = metrics.Set(ctx)

			err := handler(ctx, w, r)

			metrics.AddRequests(ctx)
			metrics.AddGoroutines(ctx)

			if err != nil {
				metrics.AddErrors(ctx)
			}

			return err
		}

		return h
	}

	return m
}
