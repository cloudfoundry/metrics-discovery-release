package app

import (
	"github.com/nats-io/nats.go"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	sd_config "github.com/prometheus/prometheus/discovery/config"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/url"
	"os"
)

type Subscriber func(queue string, callback nats.MsgHandler) (*nats.Subscription, error)

type configGenerator struct {
	scrapeConfigs map[string]config.ScrapeConfig
	path          string
	logger        *log.Logger
}

func StartConfigGeneration(subscriber Subscriber, path string, logger *log.Logger) {
	configGenerator := &configGenerator{
		scrapeConfigs: map[string]config.ScrapeConfig{},
		path:          path,
		logger:        logger,
	}

	_, err := subscriber("metrics.endpoints", configGenerator.generate)
	if err != nil {
		logger.Fatalf("failed to subscribe to metrics.endpoints: %s", err)
	}
}

func (cg *configGenerator) generate(message *nats.Msg) {
	id, scrapeConfig := cg.convertToScrapeConfig(message)
	cg.scrapeConfigs[id] = scrapeConfig

	cg.writeConfigToFile()
}

func (cg *configGenerator) convertToScrapeConfig(message *nats.Msg) (string, config.ScrapeConfig) {
	target := string(message.Data)

	targetURL, err := url.Parse(target)
	if err != nil {
		cg.logger.Printf("failed to parse target (%s) into URL: %s\n", target, err)
	}

	scrapeConfig := config.ScrapeConfig{
		JobName:     target,
		MetricsPath: targetURL.Path,
		Scheme:      targetURL.Scheme,
		ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
			StaticConfigs: []*targetgroup.Group{
				{
					Targets: []model.LabelSet{{"__address__": model.LabelValue(targetURL.Host)}},
				},
			},
		},
	}

	return target, scrapeConfig
}

func (cg *configGenerator) writeConfigToFile() {
	var scrapeConfigs []config.ScrapeConfig
	for _, scrapeConfig := range cg.scrapeConfigs {
		scrapeConfigs = append(scrapeConfigs, scrapeConfig)
	}

	data, err := yaml.Marshal(scrapeConfigs)
	if err != nil {
		cg.logger.Printf("failed to marshal scrape configs: %s\n", err)
	}

	err = ioutil.WriteFile(cg.path, data, os.ModePerm)
	if err != nil {
		cg.logger.Printf("failed to write scrape config file: %s\n", err)
	}
}
