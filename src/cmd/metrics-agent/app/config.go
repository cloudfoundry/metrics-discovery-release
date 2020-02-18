package app

import (
	"code.cloudfoundry.org/go-envstruct"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/config"
	"fmt"
	"time"
)

// Config holds the configuration for the metrics agent
type Config struct {
	MetricsExporter MetricsExporterConfig
	MetricsServer   config.MetricsServer
	GRPC            GRPCConfig

	// Scraper Certs
	ScrapeKeyPath    string `env:"SCRAPE_KEY_PATH, required, report"`
	ScrapeCertPath   string `env:"SCRAPE_CERT_PATH, required, report"`
	ScrapeCACertPath string `env:"SCRAPE_CA_CERT_PATH, required, report"`

	MetricsTargetFile string            `env:"METRICS_TARGETS_FILE, required, report"`
	ConfigGlobs       []string          `env:"CONFIG_GLOBS, report"`
	Tags              map[string]string `env:"AGENT_TAGS"`
	Addr              string            `env:"ADDR, required, report"`
	InstanceID        string            `env:"INSTANCE_ID, required, report"`
}

// MetricsExporterConfig stores the configuration for the metrics server using a PORT
// with mTLS certs.
type MetricsExporterConfig struct {
	Port                 uint16            `env:"METRICS_EXPORTER_PORT, required, report"`
	WhitelistedTimerTags []string          `env:"WHITELISTED_TIMER_TAGS, required, report"`
	DefaultLabels        map[string]string `env:"AGENT_TAGS"`

	ExpirationInterval time.Duration `env:"EXPIRATION_INTERVAL, report"`
	TimeToLive         time.Duration `env:"TTL, report"`
}

// GRPCConfig stores the configuration for the router as a server using a PORT
// with mTLS certs.
type GRPCConfig struct {
	Port     uint16 `env:"AGENT_PORT, report"`
	CAFile   string `env:"AGENT_CA_FILE_PATH, required, report"`
	CertFile string `env:"AGENT_CERT_FILE_PATH, required, report"`
	KeyFile  string `env:"AGENT_KEY_FILE_PATH, required, report"`
}

// LoadConfig will load the configuration for the forwarder agent from the
// environment. If loading the config fails for any reason this function will
// panic.
func LoadConfig() Config {
	cfg := Config{
		GRPC: GRPCConfig{
			Port: 3458,
		},
		MetricsExporter: MetricsExporterConfig{
			TimeToLive:         10 * time.Minute,
			ExpirationInterval: time.Minute,
		},
	}
	if err := envstruct.Load(&cfg); err != nil {
		panic(fmt.Sprintf("Failed to load config from environment: %s", err))
	}

	envstruct.WriteReport(&cfg)

	return cfg
}
