package main

import (
	"code.cloudfoundry.org/go-loggregator/metrics"
	"code.cloudfoundry.org/loggregator-agent/pkg/scraper"
	"code.cloudfoundry.org/metrics-discovery/cmd/metrics-agent/app"
	"log"
	_ "net/http/pprof"
	"os"
	"time"
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

	sourceIDProvider := func() []string {
		scrapeConfigs, err := scrapeConfigProvider.Configs()
		if err != nil {
			logger.Printf("unable to read scrape configs in order to blacklist source IDs: %s", err)
		}

		var sourceIDs []string
		for _, sc := range scrapeConfigs {
			sourceIDs = append(sourceIDs, sc.SourceID)
		}
		return sourceIDs
	}

	app.NewMetricsAgent(cfg, sourceIDProvider, m, logger).Run()
}
