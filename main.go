package main

import (
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tidepool-org/mailer/api"
	"github.com/tidepool-org/mailer/mailer"
	"github.com/tidepool-org/mailer/worker"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	Backend              string `envconfig:"TIDEPOOL_MAILER_BACKEND" default:"console" validate:"oneof=ses console"`
	LoggerLevel          string `envconfig:"TIDEPOOL_LOGGER_LEVEL" default:"debug" validate:"oneof=error warn info debug"`
	ServerPort           uint16 `envconfig:"TIDEPOOL_SERVICE_PORT" default:"8080" validate:"required"`
	WorkschedulerAddress string `envconfig:"TIDEPOOL_WORKSCHEDULER_ADDRESS" validate:"required"`
}

func main() {
	cfg := &Config{}
	level := zap.NewAtomicLevel()
	validate := validator.New()

	envconfig.MustProcess("", cfg)
	if err := validate.Struct(cfg); err != nil {
		log.Fatal(err)
	}
	if err := level.UnmarshalText([]byte(cfg.LoggerLevel)); err != nil {
		log.Fatal(err)
	}

	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level = level
	l, err := loggerConfig.Build()
	if err != nil {
		log.Fatal(err)
	}
	defer l.Sync()
	logger := l.Sugar()

	mlr, err := mailer.New(cfg.Backend, logger, validate)
	if err != nil {
		logger.Fatal(err.Error())
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/live", api.LiveHandler)
	mux.HandleFunc("/ready", api.ReadyHandler)

	server := http.Server{
		Addr:    fmt.Sprintf(":%v", cfg.ServerPort),
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %s", err)
		}
	}()
	logger.Infof("Server listening on port %v", cfg.ServerPort)

	wrkr := worker.New(worker.Params{
		Logger:               logger,
		Mailerr:              mlr,
		WorkschedulerAddress: "localhost:5051",
	})

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-done
		logger.Info("Received signal to shutdown server")
		cancel()
	}()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func(context.Context, *sync.WaitGroup) {
		err = wrkr.Start(ctx, wg)
		if err != nil {
			logger.Error(err)
		}
	}(ctx, wg)
	wg.Wait()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Fatalf("Server shutdown failed %s", err)
	}

	logger.Info("Server was successfully shutdown")
}
