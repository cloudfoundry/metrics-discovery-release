package target

import (
	"fmt"
	"log"
	"os"

	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/scraper"
	"gopkg.in/yaml.v2"
)

type WriterConfig struct {
	MetricsHost   string
	DefaultLabels map[string]string
	InstanceID    string
	File          string
	ScrapeConfigs []scraper.PromScraperConfig
}

func WriteFile(cfg WriterConfig, logger *log.Logger) {
	metricsExporterTarget := []string{cfg.MetricsHost}

	labels := copyMap(cfg.DefaultLabels)
	labels["instance_id"] = cfg.InstanceID

	targets := []Target{{
		Targets: metricsExporterTarget,
		Source:  fmt.Sprintf("metrics_agent_exporter__%s", cfg.InstanceID),
		Labels:  labels,
	}}

	for _, sc := range cfg.ScrapeConfigs {
		targetLabels := appendScrapeConfigLabels(labels, sc)

		targets = append(targets, Target{
			Targets: metricsExporterTarget,
			Labels:  targetLabels,
			Source:  fmt.Sprintf("%s__%s", sc.SourceID, cfg.InstanceID),
		})
	}

	writeTargets(cfg, logger, targets)
}

func writeTargets(cfg WriterConfig, logger *log.Logger, targets []Target) {
	f, err := os.Create(cfg.File)
	if err != nil {
		logger.Fatalf("unable to create metrics target file at %s: %s", cfg.File, err)
	}
	defer f.Close()

	err = yaml.NewEncoder(f).Encode(targets)
	if err != nil {
		logger.Fatalf("unable to marshal metrics target file: %s", err)
	}
}

func appendScrapeConfigLabels(labels map[string]string, sc scraper.PromScraperConfig) map[string]string {
	targetLabels := copyMap(labels)

	targetLabels["__param_id"] = sc.SourceID
	targetLabels["source_id"] = sc.SourceID

	for k, v := range sc.Labels {
		targetLabels[k] = v
	}

	return targetLabels
}

func copyMap(original map[string]string) map[string]string {
	copied := map[string]string{}

	for k, v := range original {
		copied[k] = v
	}

	return copied
}
