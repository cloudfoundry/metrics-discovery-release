package app

import (
	"code.cloudfoundry.org/go-envstruct"
	"log"
	"time"
)

type Config struct {
	Routes          []string      `env:"ROUTES,           required, report"`
	PublishInterval time.Duration `env:"PUBLISH_INTERVAL,           report"`
	NatsHosts       []string      `env:"NATS_HOSTS,       required, report"`
}

func LoadConfig(log *log.Logger) Config {
	cfg := Config{
		PublishInterval: 15 * time.Second,
	}

	if err := envstruct.Load(&cfg); err != nil {
		log.Fatal(err)
	}

	envstruct.WriteReport(&cfg)

	return cfg
}
