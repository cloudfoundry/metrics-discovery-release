package app

import (
	"log"
	"time"

	"code.cloudfoundry.org/go-envstruct"
)

type Config struct {
	NatsHosts                []string      `env:"NATS_HOSTS, required"`
	NatsCAPath               string        `env:"NATS_CA_PATH, required, report"`
	NatsCertPath             string        `env:"NATS_CERT_PATH, required, report"`
	NatsKeyPath              string        `env:"NATS_KEY_PATH, required, report"`
	ScrapeConfigFilePath     string        `env:"SCRAPE_CONFIG_FILE_PATH, required, report"`
	ConfigExpirationInterval time.Duration `env:"CONFIG_EXPIRATION_INTERVAL, report"`
	ConfigTimeToLive         time.Duration `env:"CONFIG_TTL, report"`
	WriteFrequency           time.Duration `env:"WRITE_FREQUENCY, report"`

	MetricsPort     int    `env:"METRICS_PORT, report"`
	MetricsCAPath   string `env:"METRICS_CA_PATH"`
	MetricsCertPath string `env:"METRICS_CERT_PATH"`
	MetricsKeyPath  string `env:"METRICS_KEY_PATH"`
	DebugMetrics    bool   `env:"DEBUG_METRICS, report"`
	PprofPort       uint16 `env:"PPROF_PORT, report"`
}

func LoadConfig(log *log.Logger) Config {
	cfg := Config{
		WriteFrequency:           15 * time.Second,
		ConfigExpirationInterval: 15 * time.Second,
		ConfigTimeToLive:         45 * time.Second,
	}

	if err := envstruct.Load(&cfg); err != nil {
		log.Fatal(err)
	}

	if err := envstruct.WriteReport(&cfg); err != nil {
		log.Fatal(err)
	}

	return cfg
}
