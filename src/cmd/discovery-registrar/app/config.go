package app

import (
	"log"
	"time"

	"code.cloudfoundry.org/go-envstruct"
)

type Config struct {
	PublishInterval time.Duration `env:"PUBLISH_INTERVAL,                 report"`
	NatsHosts       []string      `env:"NATS_HOSTS,             required, report"`
	NatsCAPath      string        `env:"NATS_CA_PATH,           required, report"`

	TargetsGlob           string        `env:"TARGETS_GLOB,            report"`
	TargetRefreshInterval time.Duration `env:"TARGET_REFRESH_INTERVAL, report"`

	MetricsCAPath   string `env:"METRICS_CA_PATH, required, report"`
	MetricsCertPath string `env:"METRICS_CERT_PATH, required, report"`
	MetricsKeyPath  string `env:"METRICS_KEY_PATH, required, report"`

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
