package app

import (
	"code.cloudfoundry.org/go-loggregator/metrics"
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
	"sync"
	"time"
)


type Subscriber func(queue string, callback nats.MsgHandler) (*nats.Subscription, error)

type ConfigGenerator struct {
	sync.Mutex
	scrapeConfigs            map[string]timestampedScrapeConfig
	path                     string
	logger                   *log.Logger
	configTTL                time.Duration
	configExpirationInterval time.Duration
	subscriber               Subscriber
	done                     chan struct{}
	stop                     chan struct{}
	delivered                metrics.Counter
	metrics                  metricsRegistry
}

type metricsRegistry interface {
	NewCounter(name string, opts ...metrics.MetricOption) metrics.Counter
}

type timestampedScrapeConfig struct {
	scrapeConfig config.ScrapeConfig
	ts           time.Time
}

func NewConfigGenerator(
	subscriber Subscriber,
	ttl,
	expirationInterval time.Duration,
	path string,
	m metricsRegistry,
	logger *log.Logger,
) *ConfigGenerator {
	configGenerator := &ConfigGenerator{
		scrapeConfigs:            make(map[string]timestampedScrapeConfig),
		path:                     path,
		logger:                   logger,
		configExpirationInterval: expirationInterval,
		configTTL:                ttl,
		subscriber:               subscriber,
		delivered:                m.NewCounter("delivered"),
		stop:                     make(chan struct{}),
		done:                     make(chan struct{}),
	}

	// If this doesn't happen synchronously, it could fail when the subscriber is called
	_, err := configGenerator.subscriber("metrics.endpoints", configGenerator.generate)
	if err != nil {
		configGenerator.logger.Fatalf("failed to subscribe to metrics.endpoints: %s", err)
	}

	return configGenerator
}

func (cg *ConfigGenerator) Start() {
	t := time.NewTicker(cg.configExpirationInterval)
	for {
		select {
		case <-t.C:
			cg.expireScrapeConfigs()
		case <-cg.stop:
			close(cg.done)
			return
		}
	}
}

func (cg *ConfigGenerator) Stop() {
	close(cg.stop)
	<-cg.done
}

func (cg *ConfigGenerator) generate(message *nats.Msg) {
	cg.Lock()
	defer cg.Unlock()

	cg.delivered.Add(float64(1))

	id, scrapeConfig := cg.convertToScrapeConfig(message)
	cg.scrapeConfigs[id] = timestampedScrapeConfig{
		scrapeConfig: scrapeConfig,
		ts:           time.Now(),
	}

	cg.writeConfigToFile()
}

func (cg *ConfigGenerator) convertToScrapeConfig(message *nats.Msg) (string, config.ScrapeConfig) {
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

func (cg *ConfigGenerator) writeConfigToFile() {
	var scrapeConfigs []config.ScrapeConfig
	for _, cfg := range cg.scrapeConfigs {
		scrapeConfigs = append(scrapeConfigs, cfg.scrapeConfig)
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

func (cg *ConfigGenerator) expireScrapeConfigs() {
	cg.Lock()
	defer cg.Unlock()

	for k, scrapeConfig := range cg.scrapeConfigs {
		if time.Since(scrapeConfig.ts) >= cg.configTTL {
			delete(cg.scrapeConfigs, k)
		}
	}

	cg.writeConfigToFile()
}
