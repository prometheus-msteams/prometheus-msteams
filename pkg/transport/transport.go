package transport

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/alertmanager/notify/webhook"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"

	"github.com/labstack/echo/v4"
	"github.com/prometheus-msteams/prometheus-msteams/pkg/service"
)

// Route holds the Service implementation and the Request path to serve the Service.
type Route struct {
	Service     service.Service
	RequestPath string
}

// DynamicRoute holds the Request path to generate the service based on request (e.g. path)
type DynamicRoute struct {
	ServiceGenerator func(c echo.Context) service.Service
	RequestPath      string
}

// NewServer creates the web server.
func NewServer(logger log.Logger, routes []Route, dRoutes []DynamicRoute) *echo.Echo {
	e := echo.New()
	for _, r := range routes {
		level.Debug(logger).Log("request_path_added", r.RequestPath)
		addRoute(e, r.RequestPath, r.Service, logger)
	}
	for _, r := range dRoutes {
		level.Debug(logger).Log("request_path_added", r.RequestPath)
		addContextAwareRoute(e, r.RequestPath, r.ServiceGenerator, logger)
	}
	e.HideBanner = true
	return e
}

func opencensusMiddleware() echo.MiddlewareFunc {
	return echo.WrapMiddleware(func(h http.Handler) http.Handler {
		return &ochttp.Handler{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				h.ServeHTTP(w, r)
			}),
		}
	})
}

func kitLoggerMiddleware(logger log.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func(begin time.Time) {
				res := c.Response()
				req := c.Request()
				logger.Log(
					"method", req.Method,
					"uri", req.RequestURI,
					"host", req.Host,
					"status", res.Status,
					"took", time.Since(begin),
				)
			}(time.Now())
			return next(c)
		}
	}
}

func addRoute(e *echo.Echo, p string, s service.Service, logger log.Logger) {
	e.POST(p, func(c echo.Context) error {
		return handleRoute(c, s, logger)
	},
		kitLoggerMiddleware(logger),
		opencensusMiddleware(),
	)
}

func addContextAwareRoute(e *echo.Echo, p string, w func(c echo.Context) service.Service, logger log.Logger) {
	e.POST(p, func(c echo.Context) error {
		s := w(c)
		return handleRoute(c, s, logger)
	},
		kitLoggerMiddleware(logger),
		opencensusMiddleware(),
	)
}

func handleRoute(c echo.Context, s service.Service, logger log.Logger) error {
	ctx, span := trace.StartSpan(c.Request().Context(), "alertmanager-handler")
	defer span.End()

	b, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		logger.Log("err", err)
		span.SetStatus(trace.Status{Code: 500, Message: err.Error()})
		return c.String(500, err.Error())
	}

	span.AddAttributes(trace.StringAttribute("alert", string(b)))

	var wm webhook.Message
	if err := json.Unmarshal(b, &wm); err != nil {
		logger.Log("err", err)
		span.SetStatus(trace.Status{Code: 500, Message: err.Error()})
		return c.String(500, err.Error())
	}

	prs, err := s.Post(ctx, wm)
	if err != nil {
		logger.Log("err", err)
		span.SetStatus(trace.Status{Code: 500, Message: err.Error()})
		return c.String(500, err.Error())
	}

	return c.JSON(200, prs)
}
