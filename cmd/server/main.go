package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	ocprometheus "contrib.go.opencensus.io/exporter/prometheus"
	"github.com/bzon/prometheus-msteams/pkg/card"
	"github.com/bzon/prometheus-msteams/pkg/service"
	"github.com/bzon/prometheus-msteams/pkg/transport"
	"github.com/bzon/prometheus-msteams/pkg/version"
	"github.com/labstack/echo/v4"
	stdprometheus "github.com/prometheus/client_golang/prometheus"

	"contrib.go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"

	_ "net/http/pprof"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"github.com/peterbourgon/ff"
	"gopkg.in/yaml.v2"
)

func main() {
	var (
		fs                            = flag.NewFlagSet("prometheus-msteams", flag.ExitOnError)
		logFormat                     = fs.String("log-format", "json", "json|fmt")
		debugLogs                     = fs.Bool("debug", true, "Set log level to debug mode.")
		jaegerTrace                   = fs.Bool("jaeger-trace", false, "Send traces to Jaeger.")
		jaegerAgentAddr               = fs.String("jaeger-agent", "localhost:6831", "Jaeger agent endpoint")
		httpAddr                      = fs.String("http-addr", ":2000", "HTTP listen address.")
		requestURI                    = fs.String("teams-request-uri", "", "The default request URI path where Prometheus will post to.")
		teamsWebhookURL               = fs.String("teams-incoming-webhook-url", "", "The default Microsoft Teams webhook connector.")
		templateFile                  = fs.String("template-file", "./default-message-card.tmpl", "The Microsoft Teams Message Card template file.")
		configFile                    = fs.String("config-file", "", "The connectors configuration file. WARNING! 'request-uri' and 'webhook-url' flags will be ignored if this is used.")
		httpClientIdleConnTimeout     = fs.Duration("idle-conn-timeout", 90*time.Second, "The HTTP client idle connection timeout duration.")
		httpClientTLSHandshakeTimeout = fs.Duration("tls-handshake-timeout", 30*time.Second, "The HTTP client TLS handshake timeout.")
		httpClientMaxIdleConn         = fs.Int("max-idle-conns", 100, "The HTTP client maximum number of idle connections")
	)

	if err := ff.Parse(fs, os.Args[1:], ff.WithEnvVarNoPrefix()); err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}

	// Logger.
	var logger log.Logger
	{
		switch *logFormat {
		case "json":
			logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
		case "fmt":
			logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
		default:
			fmt.Fprintf(os.Stderr, "log-format %s is not valid", *logFormat)
			os.Exit(1)
		}
		if *debugLogs {
			logger = level.NewFilter(logger, level.AllowDebug())
		} else {
			logger = level.NewFilter(logger, level.AllowInfo())
		}
		logger = log.With(logger, "ts", log.DefaultTimestamp, "caller", log.DefaultCaller)
	}

	// Tracer.
	if *jaegerTrace {
		logger.Log("message", "jaeger tracing enabled")
		je, err := jaeger.NewExporter(
			jaeger.Options{
				AgentEndpoint: *jaegerAgentAddr,
				ServiceName:   "prometheus-msteams",
			},
		)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}
		trace.RegisterExporter(je)
		trace.ApplyConfig(
			trace.Config{
				DefaultSampler: trace.AlwaysSample(),
			},
		)

	}

	// Prepare the Teams config.
	var (
		tc  TeamsConfig
		err error
	)

	// Parse the config file if defined.
	if *configFile != "" {
		tc, err = parseTeamsConfigFile(*configFile, logger)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}
	}

	// If no connectors are found,
	// use the teams-request-uri and teams-webhook-url from flags.
	if len(tc.Connectors) == 0 {
		if *requestURI == "" || *teamsWebhookURL == "" {
			logger.Log("err", "No valid connector configuration found")
			os.Exit(1)
		}
		cfgFromFlags := map[string]string{
			*requestURI: *teamsWebhookURL,
		}
		tc.Connectors = append(tc.Connectors, cfgFromFlags)
	}

	// Templated card converter setup.
	var converter card.Converter
	{
		tmpl, err := card.ParseTemplateFile(*templateFile)
		if err != nil {
			logger.Log("err", err)
		}
		converter = card.NewTemplatedCardCreator(tmpl)
		converter = card.NewCreatorLoggingMiddleware(logger, converter)
	}

	// Teams HTTP client setup.
	httpClient := &http.Client{
		Transport: &ochttp.Transport{
			Base: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:          *httpClientMaxIdleConn,
				IdleConnTimeout:       *httpClientIdleConnTimeout,
				TLSHandshakeTimeout:   *httpClientTLSHandshakeTimeout,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}

	// Routes setup.
	var routes []transport.Route
	for _, c := range tc.Connectors {
		for uri, webhook := range c {
			var r transport.Route
			r.RequestPath = uri
			r.Service = service.NewSimpleService(converter, httpClient, webhook)
			r.Service = service.NewLoggingService(logger, r.Service)
			routes = append(routes, r)
		}
	}

	pe, err := ocprometheus.NewExporter(
		ocprometheus.Options{
			Registry: stdprometheus.DefaultRegisterer.(*stdprometheus.Registry),
		},
	)
	if err != nil {
		logger.Log("err", err)
		os.Exit(1)
	}
	if err := view.Register(ocviews()...); err != nil {
		logger.Log("err", err)
		os.Exit(1)
	}

	// Prometheus msteams HTTP handler setup.
	var handler *echo.Echo
	{
		// Main app.
		handler = transport.NewServer(logger, routes...)
		// Prometheus metrics.
		handler.GET("/metrics", echo.WrapHandler(pe))
		// Pprof.
		handler.GET("/debug/pprof/*", echo.WrapHandler(http.DefaultServeMux))
		// Config.
		handler.GET("/config", func(c echo.Context) error {
			return c.JSON(200, tc.Connectors)
		})
	}

	var g run.Group
	{
		srv := http.Server{
			Addr:    *httpAddr,
			Handler: handler,
		}
		g.Add(
			func() error {
				logger.Log(
					"listen_http_addr", *httpAddr,
					"version", version.VERSION,
					"commit", version.COMMIT,
					"branch", version.BRANCH,
					"build_date", version.BUILDDATE,
				)
				return srv.ListenAndServe()
			},
			func(error) {
				if err != http.ErrServerClosed {
					if err := srv.Shutdown(context.Background()); err != nil {
						logger.Log("err", err)
					}
				}
			},
		)
	}
	{
		g.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))
	}
	logger.Log("exit", g.Run())
}

// TeamsConfig is the struct for config files
// The Connectors key is the request path for Prometheus to post
// The Connectors value is the Teams webhook url
type TeamsConfig struct {
	Connectors []map[string]string `yaml:"connectors"`
}

func parseTeamsConfigFile(f string, logger log.Logger) (TeamsConfig, error) {
	b, err := ioutil.ReadFile(f)
	if err != nil {
		return TeamsConfig{}, err
	}
	var tc TeamsConfig
	if err = yaml.Unmarshal(b, &tc); err != nil {
		return TeamsConfig{}, err
	}
	return tc, nil
}

func ocviews() []*view.View {
	keys := []tag.Key{
		ochttp.KeyClientMethod, ochttp.KeyClientStatus, ochttp.KeyClientHost, ochttp.KeyClientPath,
	}
	return []*view.View{
		&view.View{
			Name:        "http/client/sent_bytes",
			Measure:     ochttp.ClientSentBytes,
			Aggregation: view.Distribution(1024, 2048, 4096, 16384, 65536, 262144, 1048576, 4194304),
			Description: "Total bytes sent in request body (not including headers), by HTTP method and response status",
			TagKeys:     keys,
		},
		&view.View{
			Name:        "http/client/received_bytes",
			Measure:     ochttp.ClientReceivedBytes,
			Aggregation: view.Distribution(1024, 2048, 4096, 16384, 65536, 262144, 1048576, 4194304),
			Description: "Total bytes received in response bodies (not including headers but including error responses with bodies), by HTTP method and response status",
			TagKeys:     keys,
		},
		&view.View{
			Name:        "http/client/roundtrip_latency",
			Measure:     ochttp.ClientRoundtripLatency,
			Aggregation: view.Distribution(1, 2, 3, 4, 5, 6, 8, 10),
			Description: "End-to-end latency, by HTTP method and response status",
			TagKeys:     keys,
		},
		&view.View{
			Name:        "http/client/completed_count",
			Measure:     ochttp.ClientRoundtripLatency,
			Aggregation: view.Count(),
			Description: "Count of completed requests, by HTTP method and response status",
			TagKeys:     keys,
		},
	}
}
