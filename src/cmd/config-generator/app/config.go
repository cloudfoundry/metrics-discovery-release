package app

import (
	"code.cloudfoundry.org/go-envstruct"
	"log"
)

type Config struct {
	NatsHosts            []string `env:"NATS_HOSTS, required, report"`
	ScrapeConfigFilePath string   `env:"SCRAPE_CONFIG_FILE_PATH, required, report"`
}

func LoadConfig(log *log.Logger) Config {
	cfg := Config{}

	if err := envstruct.Load(&cfg); err != nil {
		log.Fatal(err)
	}

	envstruct.WriteReport(&cfg)

	return cfg
}
