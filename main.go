package main

import (
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tidepool-org/mailer/api"
	"github.com/tidepool-org/mailer/kafka"
	"github.com/tidepool-org/mailer/mailer"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Config struct {
	Backend     string `envconfig:"TIDEPOOL_MAILER_BACKEND" default:"console" validate:"oneof=ses console"`
	LoggerLevel string `envconfig:"TIDEPOOL_LOGGER_LEVEL" default:"debug" validate:"oneof=error warn info debug"`
	ServerPort  uint16  `envconfig:"TIDEPOOL_SERVICE_PORT" default:"9128" validate:"required"`
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

	kafkaConsumerConfig := &kafka.ConsumerConfig{}
	envconfig.MustProcess("", kafkaConsumerConfig)
	if err = validate.Struct(kafkaConsumerConfig); err != nil {
		logger.Fatal(err)
	}
	consumer, err := kafka.NewEmailConsumer(kafkaConsumerConfig, logger, mlr)
	if err != nil {
		logger.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/live", api.LiveHandler)
	mux.HandleFunc("/ready", api.ReadyHandler)

	server := http.Server{
		Addr: fmt.Sprintf(":%v", cfg.ServerPort),
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			logger.Fatalf("Failed to start server: %s", err)
		}
	}()
	logger.Infof("Server listening on port %v", cfg.ServerPort)

	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	go func() {
		if err := consumer.ProcessMessages(consumerCtx); err != nil {
			logger.Error(err)
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-done
	logger.Info("Received signal to shutdown server")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer func() {
		cancel()
	}()

	consumerCancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server shutdown failed %s", err)
	}

	logger.Info("Server was successfully shutdown")
}
