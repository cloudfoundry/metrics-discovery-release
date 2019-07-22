package app

import (
	"code.cloudfoundry.org/go-envstruct"
	"log"
	"time"
)

type Config struct {
	PublishInterval time.Duration `env:"PUBLISH_INTERVAL,                 report"`
	NatsHosts       []string      `env:"NATS_HOSTS,             required, report"`

	RoutesGlob           string        `env:"ROUTES_GLOB,            required, report"`
	RouteRefreshInterval time.Duration `env:"ROUTE_REFRESH_INTERVAL,           report"`
}

func LoadConfig(log *log.Logger) Config {
	cfg := Config{
		PublishInterval:      15 * time.Second,
		RouteRefreshInterval: 15 * time.Second,
	}

	if err := envstruct.Load(&cfg); err != nil {
		log.Fatal(err)
	}

	envstruct.WriteReport(&cfg)

	return cfg
}
