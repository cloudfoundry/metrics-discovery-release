package app

import (
	"code.cloudfoundry.org/go-loggregator/metrics"
	"code.cloudfoundry.org/metrics-discovery/internal/registry"
	"code.cloudfoundry.org/metrics-discovery/internal/target"
	"encoding/json"
	"github.com/nats-io/nats.go"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

type Subscriber func(queue string, callback nats.MsgHandler) (*nats.Subscription, error)

type ConfigGenerator struct {
	sync.Mutex
	path                     string
	configTTL                time.Duration
	configExpirationInterval time.Duration

	subscriber Subscriber
	done       chan struct{}
	stop       chan struct{}
	delivered  metrics.Counter
	metrics    metricsRegistry
	logger     *log.Logger

	timestampedTargets map[string]timestampedTarget
}

type metricsRegistry interface {
	NewCounter(name string, opts ...metrics.MetricOption) metrics.Counter
}

type timestampedTarget struct {
	scrapeTarget *target.Target
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
		timestampedTargets:       make(map[string]timestampedTarget),
		path:                     path,
		configTTL:                ttl,
		configExpirationInterval: expirationInterval,

		logger:     logger,
		subscriber: subscriber,
		delivered:  m.NewCounter("delivered", metrics.WithHelpText("Total number of messages successfully delivered from NATs.")),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}

	// If this doesn't happen synchronously, it could fail when the subscriber is called
	_, err := configGenerator.subscriber(registry.ScrapeTargetQueueName, configGenerator.generate)
	if err != nil {
		configGenerator.logger.Fatalf("failed to subscribe to %s: %s", registry.ScrapeTargetQueueName, err)
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

	scrapeTarget, ok := cg.unmarshalScrapeTarget(message)
	if !ok {
		return
	}

	cg.timestampedTargets[scrapeTarget.Source] = timestampedTarget{ //TODO is source real? or is it just fantasy
		scrapeTarget: scrapeTarget,
		ts:           time.Now(),
	}

	cg.writeConfigToFile()
}

func (cg *ConfigGenerator) unmarshalScrapeTarget(message *nats.Msg) (*target.Target, bool) {
	var t target.Target
	err := yaml.Unmarshal(message.Data, &t)
	if err != nil {
		cg.logger.Printf("failed to unmarshal message data: %s\n", err)
		return nil, false
	}

	return &t, true
}

func (cg *ConfigGenerator) writeConfigToFile() {
	var targets []*target.Target
	for _, cfg := range cg.timestampedTargets {
		targets = append(targets, cfg.scrapeTarget)
	}

	data, err := json.Marshal(targets)
	if err != nil {
		cg.logger.Printf("failed to marshal scrape configs: %s\n", err)
		return
	}

	err = ioutil.WriteFile(cg.path, data, os.ModePerm)
	if err != nil {
		cg.logger.Printf("failed to write scrape config file: %s\n", err)
	}
}

func (cg *ConfigGenerator) expireScrapeConfigs() {
	cg.Lock()
	defer cg.Unlock()

	for k, scrapeConfig := range cg.timestampedTargets {
		if time.Since(scrapeConfig.ts) >= cg.configTTL {
			delete(cg.timestampedTargets, k)
		}
	}

	cg.writeConfigToFile()
}
