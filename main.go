package main

import (
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tidepool-org/go-common/events"
	"github.com/tidepool-org/mailer/api"
	"github.com/tidepool-org/mailer/consumer"
	"github.com/tidepool-org/mailer/mailer"
	"github.com/tidepool-org/mailer/templates"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"log"
	"net/http"
)

type Config struct {
	Backend     mailer.Backend `envconfig:"TIDEPOOL_MAILER_BACKEND" default:"console" validate:"oneof=ses console"`
	LoggerLevel string         `envconfig:"TIDEPOOL_LOGGER_LEVEL" default:"debug" validate:"oneof=error warn info debug"`
	ServerPort  uint16         `envconfig:"TIDEPOOL_SERVICE_PORT" default:"8080" validate:"required"`
}

func provideValidator() *validator.Validate {
	return validator.New()
}

func provideConfig() (*Config, error) {
	cfg := &Config{}
	envconfig.MustProcess("", cfg)

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func provideBackend(cfg *Config) mailer.Backend {
	return cfg.Backend
}

func provideLogger(cfg *Config, lifecycle fx.Lifecycle) (*zap.SugaredLogger, error) {
	level := zap.NewAtomicLevel()

	if err := level.UnmarshalText([]byte(cfg.LoggerLevel)); err != nil {
		log.Fatal(err)
	}

	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level = level
	l, err := loggerConfig.Build()
	if err != nil {
		return nil, err
	}
	logger := l.Sugar()

	lifecycle.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(ctx context.Context) error {
			_ = l.Sync()
			return nil
		},
	})

	return logger, nil
}

type ServerParams struct {
	fx.In

	Cfg                      *Config
	Logger                   *zap.SugaredLogger
	Lifecycle                fx.Lifecycle
	TemplateSourcesHandler   http.Handler `name:"templateSourcesHandler"`
	RenderedTemplatesHandler http.Handler `name:"renderedTemplatesHandler"`
}

func provideHttpServer(params ServerParams) (*http.Server, error) {
	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())
	router.HandleFunc("/live", api.LiveHandler)
	router.HandleFunc("/ready", api.ReadyHandler)
	router.Handle("/rendered/{name}", params.RenderedTemplatesHandler)
	router.PathPrefix("/").Handler(params.TemplateSourcesHandler)

	server := http.Server{
		Addr:    fmt.Sprintf(":%v", params.Cfg.ServerPort),
		Handler: router,
	}

	return &server, nil
}

func start(eventConsumer events.EventConsumer, server *http.Server, logger *zap.SugaredLogger, lifecycle fx.Lifecycle, shutdowner fx.Shutdowner) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("Failed to start server", zap.Error(err))
					if err := shutdowner.Shutdown(); err != nil {
						logger.Error("Failed to invoke shutdowner", zap.Error(err))
					}
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := eventConsumer.Start(); err != nil && err != events.ErrConsumerStopped {
				logger.Error("Failed to start consumer", zap.Error(err))
				if err := shutdowner.Shutdown(); err != nil {
					logger.Error("Failed to invoke shutdowner", zap.Error(err))
				}
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return eventConsumer.Stop()
		},
	})
}

func main() {
	fx.New(
		fx.Provide(
			provideValidator,
			provideConfig,
			provideLogger,
			provideBackend,
			templates.NewGlobalVariables,
			templates.Load,
			mailer.New,
			consumer.New,
			fx.Annotated{
				Name:   "templateSourcesHandler",
				Target: api.TemplateSourcesHandler,
			},
			fx.Annotated{
				Name:   "renderedTemplatesHandler",
				Target: api.RenderedTemplatesHandler,
			},
			provideHttpServer,
		),
		fx.Invoke(start),
	).Run()
}
