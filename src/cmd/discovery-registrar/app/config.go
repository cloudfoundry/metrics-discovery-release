package app

import (
	"code.cloudfoundry.org/go-envstruct"
	"log"
	"time"
)

type Config struct {
	PublishInterval time.Duration `env:"PUBLISH_INTERVAL,                 report"`
	NatsHosts       []string      `env:"NATS_HOSTS,             required, report"`

	TargetsGlob           string        `env:"TARGETS_GLOB,            report"`
	TargetRefreshInterval time.Duration `env:"TARGET_REFRESH_INTERVAL, report"`

	CAFile               string            `env:"METRICS_CA_FILE_PATH, required, report"`
	CertFile             string            `env:"METRICS_CERT_FILE_PATH, required, report"`
	KeyFile              string            `env:"METRICS_KEY_FILE_PATH, required, report"`

	MetricsPort int `env:"METRICS_PORT, report"`
}

func LoadConfig(log *log.Logger) Config {
	cfg := Config{
		PublishInterval:       15 * time.Second,
		TargetRefreshInterval: 15 * time.Second,
	}

	if err := envstruct.Load(&cfg); err != nil {
		log.Fatal(err)
	}

	envstruct.WriteReport(&cfg)

	return cfg
}
