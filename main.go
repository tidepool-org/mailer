package main

import (
	"github.com/go-playground/validator/v10"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tidepool-org/mailer/mailer"
	"go.uber.org/zap"
	"log"
	"net/http"
)

type Config struct {
	Backend     string `env:"TIDEPOOL_MAILER_BACKEND",validate:"oneof=ses"`
	LoggerLevel string `env:"TIDEPOOL_LOGGER_LEVEL",default:"debug",validate:"oneof=error,warn,info,debug"`
}

func main() {
	validate := validator.New()
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	cfg := &Config{}
	err = envconfig.Process("", cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err = validate.Struct(cfg); err != nil {
		log.Fatal(err)
	}

	var backend mailer.Mailer
	switch cfg.Backend {
	case mailer.SESMailerBackendID:
		backendConfig := &mailer.SESMailerConfig{}
		if err = envconfig.Process("", backendConfig); err != nil {
			log.Fatal(err)
		}

		params := &mailer.SESMailerParams{
			Cfg: backendConfig,
			Logger: logger,
		}
		b, err := mailer.NewSESMailer(params)
		if err != nil {
			log.Fatal(err)
		}
		backend = b
	default:
		log.Fatalf("unknown mailer backend %s", cfg.Backend)
	}

	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
