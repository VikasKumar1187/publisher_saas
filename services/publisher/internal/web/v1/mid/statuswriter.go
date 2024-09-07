package mid

import (
	"context"
	"net/http"

	"github.com/vikaskumar1187/publisher_saas/services/publisher/pkg/statuswriter"
	"github.com/vikaskumar1187/publisher_saas/services/publisher/pkg/web"
	"go.uber.org/zap"
)

// ResponseWriterMiddleware wraps the ResponseWriter to debug WriteHeader calls and header writes
func ResponseWriter(log *zap.SugaredLogger) web.Middleware {
	m := func(handler web.Handler) web.Handler {
		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			recorder := &statuswriter.StatusRecorder{
				ResponseWriter: w,
				Status:         0,
				Log:            log,
			}

			err := handler(ctx, recorder, r)

			log.Infow("Request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.Status,
				"header_writes", recorder.HeaderWrites,
			)

			return err
		}
		return h
	}
	return m
}
