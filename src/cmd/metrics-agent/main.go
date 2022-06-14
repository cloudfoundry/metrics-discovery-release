package main

import (
	"log"
	"os"
	"time"

	metrics "code.cloudfoundry.org/go-metric-registry"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/scraper"
	"code.cloudfoundry.org/metrics-discovery/cmd/metrics-agent/app"
)

func main() {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	logger.Println("starting metrics-agent")
	defer logger.Println("stopping metrics-agent")

	cfg := app.LoadConfig()

	m := metrics.NewRegistry(
		logger,
		metrics.WithTLSServer(
			int(cfg.MetricsServer.Port),
			cfg.MetricsServer.CertFile,
			cfg.MetricsServer.KeyFile,
			cfg.MetricsServer.CAFile,
		),
	)

	scrapeConfigProvider := scraper.NewConfigProvider(cfg.ConfigGlobs, time.Second, logger)
	app.NewMetricsAgent(cfg, scrapeConfigProvider.Configs, m, logger).Run()
}
